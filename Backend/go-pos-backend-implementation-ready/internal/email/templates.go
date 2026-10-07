package email

// renderWelcome builds the post-registration welcome message. It is
// Arabic-first (the app's primary locale) with an English section below, so a
// new merchant understands the message whichever language they registered in.
func renderWelcome(d map[string]string) (Message, error) {
	displayName := data(d, "name", "there")
	storeName := data(d, "store", "your store")

	text := `مرحبًا بك في POS.Go

أهلاً ` + displayName + `! تم إنشاء متجرك "` + storeName + `" بنجاح.

يمكنك الآن تسجيل الدخول والبدء في البيع فورًا. اطّلع على دليل البدء السريع داخل التطبيق (الجولات التعريفية) لإعداد متجرك وإضافة أول منتج.

تطبيق الويب: https://posgo.xamltech.com/pos/
لوحة التحكم: https://api.xamltech.com/admin/
سياسة الخصوصية: https://api.xamltech.com/private

تحتاج مساعدة؟ راسلنا على ` + SupportEmail + `.

---
Welcome to POS.Go

Hi ` + displayName + `! Your store "` + storeName + `" has been created successfully.

Sign in and start selling right away. Follow the in-app guided tours to set up your store and add your first product.

Support: ` + SupportEmail

	name := esc(displayName)
	store := esc(storeName)
	body := `<p style="margin:0 0 18px;font-size:15px;line-height:1.9;color:#334155">تم إنشاء متجرك <strong>` + store + `</strong> بنجاح، ويمكنك الآن البدء في البيع فورًا. اتبع الجولات التعريفية داخل التطبيق لإعداد متجرك وإضافة أول منتج.</p>
<table role="presentation" cellpadding="0" cellspacing="0" style="margin:0 0 22px"><tr>
<td style="padding:4px 0">` + button("https://posgo.xamltech.com/pos/", "افتح تطبيق POS.Go") + `</td>
<td style="width:10px"></td>
<td style="padding:4px 0"><a href="https://api.xamltech.com/admin/" style="display:inline-block;background:#eef2ff;color:#1e3a8a;text-decoration:none;font-weight:600;padding:11px 22px;border-radius:9px;font-size:14px">لوحة التحكم</a></td>
</tr></table>
<hr style="border:none;border-top:1px solid #e5e7eb;margin:26px 0">
<div dir="ltr" style="text-align:left">
<h2 style="margin:0 0 10px;font-size:17px;color:#0f172a">Welcome to POS.Go</h2>
<p style="margin:0 0 16px;font-size:14px;line-height:1.8;color:#475569">Hi ` + name + `! Your store <strong>` + store + `</strong> has been created successfully. Sign in and start selling right away, then follow the in-app guided tours to set up your store and add your first product.</p>
</div>
<p style="margin:0;font-size:13px;line-height:1.8;color:#64748b">` + supportLine() + `</p>`

	return Message{
		From:    SupportEmail,
		Subject: "مرحبًا بك في POS.Go — Welcome to POS.Go",
		Text:    text,
		HTML:    shell("مرحبًا بك، "+displayName+"!", "نظام نقاط البيع لمتجرك", body, footerLine()),
	}, nil
}

// renderPasswordReset builds the "reset your password" mail. It carries a
// one-time link (or code) and an expiry notice; the link target is supplied by
// the caller so web + app deep links both work.
func renderPasswordReset(d map[string]string) (Message, error) {
	name := data(d, "name", "there")
	resetURL := data(d, "reset_url", "")
	code := data(d, "code", "")
	expires := data(d, "expires", "30 minutes")

	text := `مرحبًا ` + name + `,

استلمًا طلبًا لإعادة تعيين كلمة المرور الخاصة بمتجرك على POS.Go.

إذا لم تكن أنت، تجاهل هذه الرسالة ولن يتغير شيء.
`
	if resetURL != "" {
		text += `
إعادة تعيين كلمة المرور: ` + resetURL + `
`
	}
	if code != "" {
		text += `
رمز التحقق: ` + code + `
`
	}
	text += `
ينتهي الرابط/الرمز بعد ` + expires + `.

---
Hi ` + name + `,

We received a request to reset your store's POS.Go password.

If this wasn't you, ignore this message — nothing has changed.
`
	if resetURL != "" {
		text += `Reset your password: ` + resetURL + `
`
	}
	if code != "" {
		text += `Verification code: ` + code + `
`
	}
	text += `This link/code expires in ` + expires + `.

Need help? ` + SupportEmail

	n := esc(name)
	body := `<p style="margin:0 0 16px;font-size:15px;line-height:1.9;color:#334155">مرحبًا ` + n + `، استلمًا طلبًا لإعادة تعيين كلمة المرور الخاصة بمتجرك على POS.Go. إذا لم تكن أنت، تجاهل هذه الرسالة.</p>`
	if resetURL != "" {
		body += `<p style="margin:0 0 18px">` + button(resetURL, "إعادة تعيين كلمة المرور") + `</p>`
	}
	if code != "" {
		body += `<p style="margin:0 0 16px;font-size:15px;color:#334155">رمز التحقق: <strong dir="ltr" style="font-family:monospace;font-size:18px;letter-spacing:3px">` + esc(code) + `</strong></p>`
	}
	body += `<p style="margin:0 0 16px;font-size:13px;color:#64748b">ينتهي الرابط/الرمز بعد ` + esc(expires) + `.</p>
<hr style="border:none;border-top:1px solid #e5e7eb;margin:22px 0">
<div dir="ltr" style="text-align:left">
<p style="margin:0 0 8px;font-size:14px;color:#475569">Hi ` + n + `, we received a request to reset your POS.Go password. If this wasn't you, ignore this message.</p>
<p style="margin:0;font-size:13px;color:#64748b">This link/code expires in ` + esc(expires) + `. Need help? ` + SupportEmail + `</p>
</div>`

	subject := "إعادة تعيين كلمة المرور — Reset your password"
	return Message{
		From:    SupportEmail,
		Subject: subject,
		Text:    text,
		HTML:    shell("إعادة تعيين كلمة المرور", "POS.Go", body, footerLine()),
	}, nil
}

// renderVerifyEmail builds the address-verification mail (new address pending
// confirmation). The token travels only in the body; it is never stored in the
// database (only its sha256 hash).
func renderVerifyEmail(d map[string]string) (Message, error) {
	name := data(d, "name", "there")
	code := data(d, "code", "")
	expires := data(d, "expires", "24 hours")

	text := `مرحبًا ` + name + `,

استلمًا طلبًا لتأكيد عنوان بريدك الإلكتروني على POS.Go.

رمز التأكيد: ` + code + `
`
	if u := data(d, "verify_url", ""); u != "" {
		text += `رابط التأكيد: ` + u + `
`
	}
	text += `
ينتهي الرمز بعد ` + expires + `. إن لم تطلب ذلك، تجاهل هذه الرسالة.

---
Hi ` + name + `,

We received a request to confirm your POS.Go email address.

Verification code: ` + code + `
This code expires in ` + expires + `. If you didn't ask for this, ignore this message.`

	n := esc(name)
	body := `<p style="margin:0 0 16px;font-size:15px;line-height:1.9;color:#334155">مرحبًا ` + n + `، استلمًا طلبًا لتأكيد عنوان بريدك الإلكتروني على POS.Go.</p>
<p style="margin:0 0 14px;font-size:15px;color:#334155">رمز التأكيد: <strong dir="ltr" style="font-family:monospace;font-size:18px;letter-spacing:3px">` + esc(code) + `</strong></p>`
	if u := data(d, "verify_url", ""); u != "" {
		body += `<p style="margin:0 0 18px">` + button(u, "تأكيد البريد الإلكتروني") + `</p>`
	}
	body += `<p style="margin:0 0 16px;font-size:13px;color:#64748b">ينتهي الرمز بعد ` + esc(expires) + `. إن لم تطلب ذلك، تجاهل هذه الرسالة.</p>
<hr style="border:none;border-top:1px solid #e5e7eb;margin:22px 0">
<div dir="ltr" style="text-align:left">
<p style="margin:0;font-size:14px;color:#475569">Verification code: <strong dir="ltr" style="font-family:monospace;letter-spacing:2px">` + esc(code) + `</strong> — expires in ` + esc(expires) + `. If you didn't ask for this, ignore this message.</p>
</div>`

	return Message{
		From:    SupportEmail,
		Subject: "تأكيد البريد الإلكتروني — Verify your email",
		Text:    text,
		HTML:    shell("تأكيد البريد الإلكتروني", "POS.Go", body, footerLine()),
	}, nil
}

// renderInvoice builds a billing notice for an issued invoice. Amounts arrive
// pre-formatted (currency symbol included) so the template stays free of
// money math.
func renderInvoice(d map[string]string) (Message, error) {
	store := data(d, "store", "your store")
	number := data(d, "number", "")
	amount := data(d, "amount", "")
	status := data(d, "status", "")
	due := data(d, "due_date", "")

	text := `مرحبًا ` + store + `,

صدرت فاتورة جديدة على حسابك في POS.Go.

رقم الفاتورة: ` + number + `
المبلغ: ` + amount + `
الحالة: ` + status + `
`
	if due != "" {
		text += `تاريخ الاستحقاق: ` + due + `
`
	}
	text += `
روابط:
لوحة التحكم: https://api.xamltech.com/admin/
`
	if u := data(d, "invoice_url", ""); u != "" {
		text += `الفاتورة: ` + u + `
`
	}
	text += `
---
Hi ` + store + `,

A new invoice has been issued for your POS.Go account.

Invoice: ` + number + `
Amount: ` + amount + `
Status: ` + status
	if due != "" {
		text += `
Due: ` + due
	}
	text += `
Control panel: https://api.xamltech.com/admin/`

	s := esc(store)
	statusAr := esc(status)
	body := `<p style="margin:0 0 16px;font-size:15px;line-height:1.9;color:#334155">مرحبًا ` + s + `، صدرت فاتورة جديدة على حسابك في POS.Go.</p>
<table role="presentation" cellpadding="0" cellspacing="0" style="margin:0 0 20px;width:100%;background:#f8fafc;border:1px solid #e5e7eb;border-radius:10px">
<tr><td style="padding:10px 16px;font-size:14px;color:#64748b">رقم الفاتورة</td><td style="padding:10px 16px;font-size:14px;color:#0f172a;font-weight:600" dir="ltr">` + esc(number) + `</td></tr>
<tr><td style="padding:10px 16px;font-size:14px;color:#64748b;border-top:1px solid #e5e7eb">المبلغ</td><td style="padding:10px 16px;font-size:15px;color:#0f172a;font-weight:700;border-top:1px solid #e5e7eb" dir="ltr">` + esc(amount) + `</td></tr>
<tr><td style="padding:10px 16px;font-size:14px;color:#64748b;border-top:1px solid #e5e7eb">الحالة</td><td style="padding:10px 16px;font-size:14px;color:#0f172a;border-top:1px solid #e5e7eb">` + statusAr + `</td></tr>`
	if due != "" {
		body += `
<tr><td style="padding:10px 16px;font-size:14px;color:#64748b;border-top:1px solid #e5e7eb">تاريخ الاستحقاق</td><td style="padding:10px 16px;font-size:14px;color:#0f172a;border-top:1px solid #e5e7eb" dir="ltr">` + esc(due) + `</td></tr>`
	}
	body += `</table>
<p style="margin:0 0 10px">` + button("https://api.xamltech.com/admin/", "افتح لوحة التحكم") + `</p>
<p style="margin:0;font-size:13px;color:#64748b">` + supportLine() + `</p>`

	subject := sanitizeSubject("فاتورة جديدة — New invoice")
	if n := sanitizeSubject(number); n != "" {
		subject += " " + n
	}
	return Message{
		From:    SupportEmail,
		Subject: subject,
		Text:    text,
		HTML:    shell("فاتورة جديدة", "POS.Go", body, footerLine()),
	}, nil
}

// renderNotification is the generic operator→merchant notice (e.g. "we
// reacted your account", "your trial ends soon"). title/body are supplied by
// the workflow so this stays reusable.
func renderNotification(d map[string]string) (Message, error) {
	name := data(d, "name", "there")
	title := data(d, "title", "إشعار من POS.Go")
	body := data(d, "body", "")

	text := `مرحبًا ` + name + `,

` + title + `

` + body + `

---
Hi ` + name + `,

` + title + `

` + body + `

Support: ` + SupportEmail

	n := esc(name)
	htmlBody := `<p style="margin:0 0 14px;font-size:15px;line-height:1.9;color:#334155">مرحبًا ` + n + `،</p>
<h2 style="margin:0 0 14px;font-size:18px;color:#0f172a">` + esc(title) + `</h2>
<div style="font-size:15px;line-height:1.9;color:#334155;white-space:pre-wrap">` + esc(body) + `</div>
<hr style="border:none;border-top:1px solid #e5e7eb;margin:22px 0">
<div dir="ltr" style="text-align:left">
<p style="margin:0;font-size:14px;color:#475569">Hi ` + n + ` — ` + esc(title) + `</p>
<p style="margin:6px 0 0;font-size:13px;color:#64748b">` + esc(body) + `</p>
</div>`

	subject := sanitizeSubject(title)
	if subject == "" {
		subject = "إشعار من POS.Go"
	}

	return Message{
		From:    SupportEmail,
		Subject: subject,
		Text:    text,
		HTML:    shell(title, "POS.Go", htmlBody, footerLine()),
	}, nil
}

// renderAccountNotice covers the lifecycle mail an admin triggers from the
// control panel: activation, stop, reactivation, suspension.
func renderAccountNotice(d map[string]string) (Message, error) {
	// Reuse the notification shape with a stronger header.
	if _, ok := d["title"]; !ok {
		d = copyWith(d, "title", "تحديث حالة حسابك")
	}
	return renderNotification(d)
}

// renderSubscriptionEnded tells a merchant their subscription is no longer
// active and how to renew.
func renderSubscriptionEnded(d map[string]string) (Message, error) {
	name := data(d, "name", "there")
	reason := data(d, "reason", "")
	renewURL := data(d, "renew_url", "https://api.xamltech.com/admin/")

	text := `مرحبًا ` + name + `,

انتهت/توقفت اشتراكاتك في POS.Go. لن تتم معالجة عمليات البيع الجديدة حتى التجديد.
`
	if reason != "" {
		text += `السبب: ` + reason + `
`
	}
	text += `تجديد الاشتراك: ` + renewURL + `
للمساعدة: ` + SupportEmail + `

---
Hi ` + name + `,

Your POS.Go subscription has ended. New sales will not be processed until you renew.
`
	if reason != "" {
		text += `Reason: ` + reason + `
`
	}
	text += `Renew: ` + renewURL + `
Support: ` + SupportEmail

	n := esc(name)
	body := `<p style="margin:0 0 14px;font-size:15px;line-height:1.9;color:#334155">مرحبًا ` + n + `، انتهت/توقفت اشتراكاتك في POS.Go. لن تتم معالجة عمليات البيع الجديدة حتى التجديد.</p>`
	if reason != "" {
		body += `<p style="margin:0 0 14px;font-size:14px;color:#64748b">السبب: ` + esc(reason) + `</p>`
	}
	body += `<p style="margin:0 0 18px">` + button(renewURL, "تجديد الاشتراك") + `</p>
<hr style="border:none;border-top:1px solid #e5e7eb;margin:22px 0">
<div dir="ltr" style="text-align:left">
<p style="margin:0;font-size:14px;color:#475569">Your POS.Go subscription has ended. Renew to resume sales.</p>
<p style="margin:6px 0 0;font-size:13px;color:#64748b">Support: ` + SupportEmail + `</p>
</div>`

	return Message{
		From:    SupportEmail,
		Subject: "انتهاء الاشتراك — Subscription ended",
		Text:    text,
		HTML:    shell("انتهاء الاشتراك", "POS.Go", body, footerLine()),
	}, nil
}

// copyWith returns d with key set to value without mutating the input map.
func copyWith(d map[string]string, key, value string) map[string]string {
	out := make(map[string]string, len(d)+1)
	for k, v := range d {
		out[k] = v
	}
	out[key] = value
	return out
}
