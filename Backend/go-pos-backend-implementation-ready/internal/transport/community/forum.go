package community

import (
	"errors"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// ForumCategory is the public wire shape of a forum board.
type ForumCategory struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
	SortOrder   int    `json:"sort_order"`
	IsActive    bool   `json:"is_active"`
	PostCount   int    `json:"post_count"`
	CreatedAt   string `json:"created_at"`
}

// ForumPost is the public wire shape of one thread, with the list-view
// aggregates (counts + viewer state) the client renders inline.
type ForumPost struct {
	ID           string `json:"id"`
	CategoryID   string `json:"category_id"`
	CategoryName string `json:"category_name"`
	AuthorID     string `json:"author_id"`
	AuthorName   string `json:"author_name"`
	Title        string `json:"title"`
	Body         string `json:"body"`
	IsPinned     bool   `json:"is_pinned"`
	IsClosed     bool   `json:"is_closed"`
	CommentCount int    `json:"comment_count"`
	LikeCount    int    `json:"like_count"`
	LikedByMe    bool   `json:"liked_by_me"`
	IsAuthor     bool   `json:"is_author"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

// ForumComment is the public wire shape of one reply under a thread.
type ForumComment struct {
	ID         string `json:"id"`
	PostID     string `json:"post_id"`
	AuthorID   string `json:"author_id"`
	AuthorName string `json:"author_name"`
	Body       string `json:"body"`
	CreatedAt  string `json:"created_at"`
}

type createForumCategoryRequest struct {
	Name        string `json:"name" binding:"required"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
	SortOrder   *int   `json:"sort_order"`
}

type updateForumCategoryRequest struct {
	Name        *string `json:"name"`
	Slug        *string `json:"slug"`
	Description *string `json:"description"`
	SortOrder   *int    `json:"sort_order"`
	IsActive    *bool   `json:"is_active"`
}

type createForumPostRequest struct {
	CategoryID string `json:"category_id" binding:"required"`
	Title      string `json:"title" binding:"required"`
	Body       string `json:"body"`
}

type updateForumPostRequest struct {
	Title    *string `json:"title"`
	Body     *string `json:"body"`
	IsPinned *bool   `json:"is_pinned"`
	IsClosed *bool   `json:"is_closed"`
}

type createForumCommentRequest struct {
	Body string `json:"body" binding:"required"`
}

// forumSlugPattern bounds a client-supplied slug; auto-slugs from names pass
// the same shape (1-100 chars, alnum ends so a one-letter name still works).
var forumSlugPattern = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{0,98}[a-z0-9])?$`)

// forumSlug derives a URL-ish slug from a category name (fallback "board").
func forumSlug(name string) string {
	var b strings.Builder
	lastDash := false
	for _, r := range strings.ToLower(strings.TrimSpace(name)) {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			b.WriteRune(r)
			lastDash = false
		default:
			if !lastDash && b.Len() > 0 {
				b.WriteByte('-')
				lastDash = true
			}
		}
	}
	slug := strings.Trim(b.String(), "-")
	if slug == "" {
		return "board"
	}
	if len(slug) > 100 {
		slug = strings.Trim(slug[:100], "-")
	}
	return slug
}

func (h *Handler) listForumCategories(c *gin.Context) {
	claims, ok := h.authenticate(c)
	if !ok {
		return
	}
	if !h.can(c, claims.Role, "community", "read") {
		return
	}
	if h.pool == nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	includeInactive := c.Query("all") == "1" || c.Query("all") == "true"

	ctx := c.Request.Context()
	tx, err := h.pool.Begin(ctx)
	if err != nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err = tx.Exec(ctx, "SELECT set_config('app.current_tenant', $1, true)", claims.TenantID); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to establish tenant context")
		return
	}

	where := "TRUE"
	if !includeInactive {
		where += " AND c.is_active = TRUE"
	}
	rows, err := tx.Query(ctx, `
		SELECT c.id::text, c.name, c.slug, c.description, c.sort_order, c.is_active,
			c.created_at::text,
			(SELECT COUNT(*) FROM forum_posts p WHERE p.category_id = c.id)::int
		FROM forum_categories c
		WHERE `+where+`
		ORDER BY c.sort_order, c.name`)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load categories")
		return
	}
	defer rows.Close()

	items := make([]ForumCategory, 0)
	for rows.Next() {
		var item ForumCategory
		if err := rows.Scan(&item.ID, &item.Name, &item.Slug, &item.Description,
			&item.SortOrder, &item.IsActive, &item.CreatedAt, &item.PostCount); err != nil {
			writeError(c, http.StatusInternalServerError, "internal_error", "unable to load categories")
			return
		}
		items = append(items, item)
	}
	if err = rows.Err(); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load categories")
		return
	}
	if err = tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load categories")
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"data": items,
		"meta": gin.H{"request_id": c.GetString("request_id"), "total": len(items)},
	})
}

func (h *Handler) createForumCategory(c *gin.Context) {
	claims, ok := h.authenticate(c)
	if !ok {
		return
	}
	if !h.can(c, claims.Role, "community", "write") {
		return
	}
	var request createForumCategoryRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "invalid category request")
		return
	}
	name := strings.TrimSpace(request.Name)
	if name == "" || len(name) > 100 {
		writeError(c, http.StatusBadRequest, "validation_error", "name is required and must be at most 100 characters")
		return
	}
	if len(request.Description) > 500 {
		writeError(c, http.StatusBadRequest, "validation_error", "description is too long")
		return
	}
	slug := strings.TrimSpace(request.Slug)
	if slug == "" {
		slug = forumSlug(name)
	}
	if !forumSlugPattern.MatchString(slug) {
		writeError(c, http.StatusBadRequest, "validation_error", "slug must be kebab-case (a-z, 0-9, dashes)")
		return
	}
	sortOrder := 0
	if request.SortOrder != nil {
		sortOrder = *request.SortOrder
	}
	if h.pool == nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	ctx := c.Request.Context()
	tx, err := h.pool.Begin(ctx)
	if err != nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err = tx.Exec(ctx, "SELECT set_config('app.current_tenant', $1, true)", claims.TenantID); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to establish tenant context")
		return
	}

	categoryID := uuid.New()
	var createdAt string
	err = tx.QueryRow(ctx, `
		INSERT INTO forum_categories (id, tenant_id, name, slug, description, sort_order)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING created_at::text`,
		categoryID, claims.TenantID, name, slug,
		strings.TrimSpace(request.Description), sortOrder).Scan(&createdAt)
	if err != nil {
		if strings.Contains(err.Error(), "unique") {
			writeError(c, http.StatusConflict, "category_taken", "category name already exists")
			return
		}
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to save category")
		return
	}
	if err = tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to save category")
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"data": ForumCategory{
			ID: categoryID.String(), Name: name, Slug: slug,
			Description: strings.TrimSpace(request.Description),
			SortOrder:   sortOrder, IsActive: true, CreatedAt: createdAt,
		},
		"meta": gin.H{"request_id": c.GetString("request_id")},
	})
}

func (h *Handler) updateForumCategory(c *gin.Context) {
	claims, ok := h.authenticate(c)
	if !ok {
		return
	}
	if !h.can(c, claims.Role, "community", "write") {
		return
	}
	categoryID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "invalid category id")
		return
	}
	var request updateForumCategoryRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "invalid category request")
		return
	}
	if h.pool == nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	ctx := c.Request.Context()
	tx, err := h.pool.Begin(ctx)
	if err != nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err = tx.Exec(ctx, "SELECT set_config('app.current_tenant', $1, true)", claims.TenantID); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to establish tenant context")
		return
	}

	var current ForumCategory
	err = tx.QueryRow(ctx, `
		SELECT id::text, name, slug, description, sort_order, is_active, created_at::text
		FROM forum_categories WHERE id = $1`, categoryID).Scan(
		&current.ID, &current.Name, &current.Slug, &current.Description,
		&current.SortOrder, &current.IsActive, &current.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(c, http.StatusNotFound, "category_not_found", "category not found")
		return
	}
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load category")
		return
	}
	if request.Name != nil {
		current.Name = strings.TrimSpace(*request.Name)
	}
	if request.Slug != nil {
		current.Slug = strings.TrimSpace(*request.Slug)
	}
	if request.Description != nil {
		current.Description = strings.TrimSpace(*request.Description)
	}
	if request.SortOrder != nil {
		current.SortOrder = *request.SortOrder
	}
	if request.IsActive != nil {
		current.IsActive = *request.IsActive
	}
	if current.Name == "" || len(current.Name) > 100 || len(current.Description) > 500 {
		writeError(c, http.StatusBadRequest, "validation_error", "category fields are invalid")
		return
	}
	if current.Slug != "" && !forumSlugPattern.MatchString(current.Slug) {
		writeError(c, http.StatusBadRequest, "validation_error", "slug must be kebab-case (a-z, 0-9, dashes)")
		return
	}

	err = tx.QueryRow(ctx, `
		UPDATE forum_categories SET name = $1, slug = $2, description = $3,
			sort_order = $4, is_active = $5
		WHERE id = $6
		RETURNING created_at::text`,
		current.Name, current.Slug, current.Description,
		current.SortOrder, current.IsActive, categoryID).Scan(&current.CreatedAt)
	if err != nil {
		if strings.Contains(err.Error(), "unique") {
			writeError(c, http.StatusConflict, "category_taken", "category name already exists")
			return
		}
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to update category")
		return
	}
	if err = tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to update category")
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"data": current,
		"meta": gin.H{"request_id": c.GetString("request_id")},
	})
}

// forumPostSelectSQL is the shared post projection: the row plus the viewer's
// aggregates (counts, liked_by_me, is_author). $1 is always the viewer id.
const forumPostSelectSQL = `
	SELECT p.id::text, p.category_id::text, c.name, p.author_id::text,
		COALESCE(u.display_name, ''), p.title, p.body, p.is_pinned, p.is_closed,
		p.created_at::text, p.updated_at::text,
		(SELECT COUNT(*) FROM forum_comments fc WHERE fc.post_id = p.id)::int,
		(SELECT COUNT(*) FROM forum_post_likes fl WHERE fl.post_id = p.id)::int,
		EXISTS (SELECT 1 FROM forum_post_likes fl2 WHERE fl2.post_id = p.id AND fl2.user_id = $1::uuid),
		p.author_id = $1::uuid
	FROM forum_posts p
	JOIN forum_categories c ON c.id = p.category_id
	LEFT JOIN users u ON u.id = p.author_id`

func scanForumPost(row pgx.Row, post *ForumPost) error {
	return row.Scan(&post.ID, &post.CategoryID, &post.CategoryName, &post.AuthorID,
		&post.AuthorName, &post.Title, &post.Body, &post.IsPinned, &post.IsClosed,
		&post.CreatedAt, &post.UpdatedAt, &post.CommentCount, &post.LikeCount,
		&post.LikedByMe, &post.IsAuthor)
}

func (h *Handler) listForumPosts(c *gin.Context) {
	claims, ok := h.authenticate(c)
	if !ok {
		return
	}
	if !h.can(c, claims.Role, "community", "read") {
		return
	}
	// Parse every query bound before touching the pool so bad input answers
	// 400 without a database round-trip.
	page, limit := pageLimits(c)
	q := strings.TrimSpace(c.Query("q"))
	sort := strings.TrimSpace(c.Query("sort"))
	switch sort {
	case "", "latest", "top", "pinned":
	default:
		writeError(c, http.StatusBadRequest, "validation_error", "sort must be latest, top or pinned")
		return
	}
	// The WHERE clause always references $1 (the viewer) so both the count and
	// list queries bind the same argument list — filters start at $2. Without
	// that anchor an unfiltered count would bind a $1 Postgres never typed.
	where := "$1::uuid IS NOT NULL"
	args := []interface{}{claims.UserID}
	next := 2
	if raw := strings.TrimSpace(c.Query("category_id")); raw != "" {
		categoryID, err := uuid.Parse(raw)
		if err != nil {
			writeError(c, http.StatusBadRequest, "validation_error", "invalid category id")
			return
		}
		where += " AND p.category_id = $" + strconv.Itoa(next)
		args = append(args, categoryID)
		next++
	}
	if q != "" {
		where += " AND (p.title ILIKE $" + strconv.Itoa(next) + " OR p.body ILIKE $" + strconv.Itoa(next) + ")"
		args = append(args, "%"+q+"%")
		next++
	}
	if h.pool == nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	order := "p.created_at DESC"
	switch sort {
	case "top":
		order = "(SELECT COUNT(*) FROM forum_post_likes fl WHERE fl.post_id = p.id) DESC, p.created_at DESC"
	case "pinned":
		order = "p.is_pinned DESC, p.created_at DESC"
	}

	ctx := c.Request.Context()
	tx, err := h.pool.Begin(ctx)
	if err != nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err = tx.Exec(ctx, "SELECT set_config('app.current_tenant', $1, true)", claims.TenantID); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to establish tenant context")
		return
	}

	var total int64
	if err = tx.QueryRow(ctx, `
		SELECT COUNT(*) FROM forum_posts p WHERE `+where, args...).Scan(&total); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to count posts")
		return
	}

	listArgs := append(append([]interface{}{}, args...), limit, (page-1)*limit)
	rows, err := tx.Query(ctx, forumPostSelectSQL+`
	WHERE `+where+` ORDER BY `+order+`
	LIMIT $`+strconv.Itoa(next)+` OFFSET $`+strconv.Itoa(next+1), listArgs...)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load posts")
		return
	}
	defer rows.Close()

	items := make([]ForumPost, 0)
	for rows.Next() {
		var item ForumPost
		if err := rows.Scan(&item.ID, &item.CategoryID, &item.CategoryName, &item.AuthorID,
			&item.AuthorName, &item.Title, &item.Body, &item.IsPinned, &item.IsClosed,
			&item.CreatedAt, &item.UpdatedAt, &item.CommentCount, &item.LikeCount,
			&item.LikedByMe, &item.IsAuthor); err != nil {
			writeError(c, http.StatusInternalServerError, "internal_error", "unable to load posts")
			return
		}
		items = append(items, item)
	}
	if err = rows.Err(); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load posts")
		return
	}
	if err = tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load posts")
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"data": items,
		"meta": gin.H{"request_id": c.GetString("request_id"), "page": page, "limit": limit, "total": total},
	})
}

func (h *Handler) createForumPost(c *gin.Context) {
	claims, ok := h.authenticate(c)
	if !ok {
		return
	}
	if !h.can(c, claims.Role, "forum", "write") {
		return
	}
	var request createForumPostRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "invalid post request")
		return
	}
	categoryID, err := uuid.Parse(request.CategoryID)
	if err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "invalid category id")
		return
	}
	title := strings.TrimSpace(request.Title)
	if title == "" || len(title) > 200 {
		writeError(c, http.StatusBadRequest, "validation_error", "title is required and must be at most 200 characters")
		return
	}
	body := strings.TrimSpace(request.Body)
	if len(body) > 10000 {
		writeError(c, http.StatusBadRequest, "validation_error", "body is too long")
		return
	}
	if h.pool == nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	ctx := c.Request.Context()
	tx, err := h.pool.Begin(ctx)
	if err != nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err = tx.Exec(ctx, "SELECT set_config('app.current_tenant', $1, true)", claims.TenantID); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to establish tenant context")
		return
	}

	var categoryName string
	var categoryActive bool
	err = tx.QueryRow(ctx, `SELECT name, is_active FROM forum_categories WHERE id = $1`, categoryID).
		Scan(&categoryName, &categoryActive)
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && !categoryActive) {
		writeError(c, http.StatusNotFound, "category_not_found", "category not found")
		return
	}
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to verify category")
		return
	}

	postID := uuid.New()
	var authorName, createdAt, updatedAt string
	err = tx.QueryRow(ctx, `
		INSERT INTO forum_posts (id, tenant_id, category_id, author_id, title, body)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING created_at::text, updated_at::text`,
		postID, claims.TenantID, categoryID, claims.UserID, title, body).
		Scan(&createdAt, &updatedAt)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to save post")
		return
	}
	_ = tx.QueryRow(ctx, `SELECT display_name FROM users WHERE id = $1`, claims.UserID).Scan(&authorName)
	if err = tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to save post")
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"data": ForumPost{
			ID: postID.String(), CategoryID: categoryID.String(), CategoryName: categoryName,
			AuthorID: claims.UserID, AuthorName: authorName,
			Title: title, Body: body, IsAuthor: true,
			CreatedAt: createdAt, UpdatedAt: updatedAt,
		},
		"meta": gin.H{"request_id": c.GetString("request_id")},
	})
}

func (h *Handler) getForumPost(c *gin.Context) {
	claims, ok := h.authenticate(c)
	if !ok {
		return
	}
	if !h.can(c, claims.Role, "community", "read") {
		return
	}
	postID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "invalid post id")
		return
	}
	if h.pool == nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	ctx := c.Request.Context()
	tx, err := h.pool.Begin(ctx)
	if err != nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err = tx.Exec(ctx, "SELECT set_config('app.current_tenant', $1, true)", claims.TenantID); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to establish tenant context")
		return
	}

	var post ForumPost
	row := tx.QueryRow(ctx, forumPostSelectSQL+` WHERE p.id = $2::uuid`, claims.UserID, postID)
	if err := scanForumPost(row, &post); errors.Is(err, pgx.ErrNoRows) {
		writeError(c, http.StatusNotFound, "post_not_found", "post not found")
		return
	} else if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load post")
		return
	}

	commentRows, err := tx.Query(ctx, `
		SELECT fc.id::text, fc.post_id::text, fc.author_id::text, COALESCE(u.display_name, ''),
			fc.body, fc.created_at::text
		FROM forum_comments fc
		LEFT JOIN users u ON u.id = fc.author_id
		WHERE fc.post_id = $1
		ORDER BY fc.created_at`, postID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load comments")
		return
	}
	defer commentRows.Close()
	comments := make([]ForumComment, 0)
	for commentRows.Next() {
		var item ForumComment
		if err := commentRows.Scan(&item.ID, &item.PostID, &item.AuthorID, &item.AuthorName,
			&item.Body, &item.CreatedAt); err != nil {
			writeError(c, http.StatusInternalServerError, "internal_error", "unable to load comments")
			return
		}
		comments = append(comments, item)
	}
	if err := commentRows.Err(); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load comments")
		return
	}
	if err = tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load post")
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"data":     post,
		"comments": comments,
		"meta":     gin.H{"request_id": c.GetString("request_id")},
	})
}

func (h *Handler) updateForumPost(c *gin.Context) {
	claims, ok := h.authenticate(c)
	if !ok {
		return
	}
	postID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "invalid post id")
		return
	}
	var request updateForumPostRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "invalid post request")
		return
	}
	if h.pool == nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	ctx := c.Request.Context()
	tx, err := h.pool.Begin(ctx)
	if err != nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err = tx.Exec(ctx, "SELECT set_config('app.current_tenant', $1, true)", claims.TenantID); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to establish tenant context")
		return
	}

	var current ForumPost
	row := tx.QueryRow(ctx, forumPostSelectSQL+` WHERE p.id = $2::uuid`, claims.UserID, postID)
	if err := scanForumPost(row, &current); errors.Is(err, pgx.ErrNoRows) {
		writeError(c, http.StatusNotFound, "post_not_found", "post not found")
		return
	} else if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load post")
		return
	}

	// Author may edit content; moderation flags (pin/close) ride on
	// community.write exactly like editing someone else's post.
	moderating := request.IsPinned != nil || request.IsClosed != nil
	if current.AuthorID != claims.UserID {
		if !h.can(c, claims.Role, "community", "write") {
			return
		}
	} else if moderating && !h.can(c, claims.Role, "community", "write") {
		return
	}

	title, body := current.Title, current.Body
	if request.Title != nil {
		title = strings.TrimSpace(*request.Title)
	}
	if request.Body != nil {
		body = strings.TrimSpace(*request.Body)
	}
	if title == "" || len(title) > 200 || len(body) > 10000 {
		writeError(c, http.StatusBadRequest, "validation_error", "post fields are invalid")
		return
	}
	isPinned, isClosed := current.IsPinned, current.IsClosed
	if request.IsPinned != nil {
		isPinned = *request.IsPinned
	}
	if request.IsClosed != nil {
		isClosed = *request.IsClosed
	}

	var post ForumPost
	row = tx.QueryRow(ctx, `
		UPDATE forum_posts SET title = $1, body = $2, is_pinned = $3, is_closed = $4,
			updated_at = NOW()
		WHERE id = $5
		RETURNING id::text`, title, body, isPinned, isClosed, postID)
	var updatedID string
	if err := row.Scan(&updatedID); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to update post")
		return
	}
	row = tx.QueryRow(ctx, forumPostSelectSQL+` WHERE p.id = $2::uuid`, claims.UserID, postID)
	if err := scanForumPost(row, &post); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load post")
		return
	}
	if err = tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to update post")
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"data": post,
		"meta": gin.H{"request_id": c.GetString("request_id")},
	})
}

func (h *Handler) deleteForumPost(c *gin.Context) {
	claims, ok := h.authenticate(c)
	if !ok {
		return
	}
	postID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "invalid post id")
		return
	}
	if h.pool == nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	ctx := c.Request.Context()
	tx, err := h.pool.Begin(ctx)
	if err != nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err = tx.Exec(ctx, "SELECT set_config('app.current_tenant', $1, true)", claims.TenantID); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to establish tenant context")
		return
	}

	var authorID string
	if err := tx.QueryRow(ctx, `SELECT author_id::text FROM forum_posts WHERE id = $1`, postID).Scan(&authorID); errors.Is(err, pgx.ErrNoRows) {
		writeError(c, http.StatusNotFound, "post_not_found", "post not found")
		return
	} else if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load post")
		return
	}
	if authorID != claims.UserID {
		if !h.can(c, claims.Role, "community", "write") {
			return
		}
	}

	tag, err := tx.Exec(ctx, `DELETE FROM forum_posts WHERE id = $1`, postID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to delete post")
		return
	}
	if tag.RowsAffected() == 0 {
		writeError(c, http.StatusNotFound, "post_not_found", "post not found")
		return
	}
	if err = tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to delete post")
		return
	}
	c.JSON(http.StatusOK, gin.H{"meta": gin.H{"request_id": c.GetString("request_id")}})
}

func (h *Handler) createForumComment(c *gin.Context) {
	claims, ok := h.authenticate(c)
	if !ok {
		return
	}
	if !h.can(c, claims.Role, "forum", "write") {
		return
	}
	postID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "invalid post id")
		return
	}
	var request createForumCommentRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "invalid comment request")
		return
	}
	body := strings.TrimSpace(request.Body)
	if body == "" || len(body) > 5000 {
		writeError(c, http.StatusBadRequest, "validation_error", "body is required and must be at most 5000 characters")
		return
	}
	if h.pool == nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	ctx := c.Request.Context()
	tx, err := h.pool.Begin(ctx)
	if err != nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err = tx.Exec(ctx, "SELECT set_config('app.current_tenant', $1, true)", claims.TenantID); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to establish tenant context")
		return
	}

	var isClosed bool
	if err := tx.QueryRow(ctx, `SELECT is_closed FROM forum_posts WHERE id = $1`, postID).Scan(&isClosed); errors.Is(err, pgx.ErrNoRows) {
		writeError(c, http.StatusNotFound, "post_not_found", "post not found")
		return
	} else if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to verify post")
		return
	}
	if isClosed {
		writeError(c, http.StatusConflict, "post_closed", "post is closed for new comments")
		return
	}

	commentID := uuid.New()
	var authorName, createdAt string
	if err = tx.QueryRow(ctx, `
		INSERT INTO forum_comments (id, tenant_id, post_id, author_id, body)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING created_at::text`,
		commentID, claims.TenantID, postID, claims.UserID, body).Scan(&createdAt); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to save comment")
		return
	}
	_ = tx.QueryRow(ctx, `SELECT display_name FROM users WHERE id = $1`, claims.UserID).Scan(&authorName)
	if err = tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to save comment")
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"data": ForumComment{
			ID: commentID.String(), PostID: postID.String(),
			AuthorID: claims.UserID, AuthorName: authorName,
			Body: body, CreatedAt: createdAt,
		},
		"meta": gin.H{"request_id": c.GetString("request_id")},
	})
}

func (h *Handler) toggleForumLike(c *gin.Context) {
	claims, ok := h.authenticate(c)
	if !ok {
		return
	}
	if !h.can(c, claims.Role, "forum", "write") {
		return
	}
	postID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "invalid post id")
		return
	}
	if h.pool == nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	ctx := c.Request.Context()
	tx, err := h.pool.Begin(ctx)
	if err != nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err = tx.Exec(ctx, "SELECT set_config('app.current_tenant', $1, true)", claims.TenantID); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to establish tenant context")
		return
	}

	var postExists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM forum_posts WHERE id = $1)`, postID).Scan(&postExists); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to verify post")
		return
	}
	if !postExists {
		writeError(c, http.StatusNotFound, "post_not_found", "post not found")
		return
	}

	// Toggle: a prior row is removed, otherwise one is added. The composite
	// PK keeps concurrent duplicates out; re-clicking flips the state.
	liked := false
	tag, err := tx.Exec(ctx, `DELETE FROM forum_post_likes WHERE post_id = $1 AND user_id = $2`, postID, claims.UserID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to update like")
		return
	}
	if tag.RowsAffected() == 0 {
		if _, err := tx.Exec(ctx, `
			INSERT INTO forum_post_likes (tenant_id, post_id, user_id)
			VALUES ($1, $2, $3)
			ON CONFLICT DO NOTHING`,
			claims.TenantID, postID, claims.UserID); err != nil {
			writeError(c, http.StatusInternalServerError, "internal_error", "unable to update like")
			return
		}
		liked = true
	}
	var likeCount int
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM forum_post_likes WHERE post_id = $1`, postID).Scan(&likeCount); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to count likes")
		return
	}
	if err = tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to update like")
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{"liked": liked, "like_count": likeCount},
		"meta": gin.H{"request_id": c.GetString("request_id")},
	})
}
