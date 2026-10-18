import { useState, type ReactNode } from "react";
import { arrOf, cv, str, type Lang, type Raw } from "../content";
import { localizeHref } from "../i18n";

// --- small helpers ----------------------------------------------------------

function SectionHead({
  eyebrow,
  title,
  sub,
}: {
  eyebrow?: string;
  title?: string;
  sub?: string;
}) {
  if (!eyebrow && !title && !sub) return null;
  return (
    <div className="section-head">
      {eyebrow ? <span className="section-label">{eyebrow}</span> : null}
      {title ? <h2 className="section-title">{title}</h2> : null}
      {sub ? <p className="section-sub">{sub}</p> : null}
    </div>
  );
}

function Btn({
  href,
  label,
  style,
  big,
  external,
}: {
  href: string;
  label: string;
  style?: string;
  big?: boolean;
  external?: boolean;
}) {
  const variant = style && style !== "primary" ? `btn-${style}` : "btn-primary";
  return (
    <a
      className={`btn ${variant}${big ? " btn-lg" : ""}`}
      href={href}
      {...(external ? { target: "_blank", rel: "noopener" } : {})}
    >
      {label}
    </a>
  );
}

function Ticks({ points }: { points: string[] }) {
  if (!points.length) return null;
  return (
    <ul className="ticks">
      {points.map((p, i) => (
        <li key={i}>{p}</li>
      ))}
    </ul>
  );
}

// --- hero -------------------------------------------------------------------

function Hero({ data, lang }: { data: Raw; lang: Lang }) {
  const points = arrOf(data.points).map(str);
  const ctas = arrOf(data.ctas);
  const stats = arrOf(data.stats);
  const media = (data.media ?? null) as Raw | null;
  const title = cv(data, "title");
  const accent = cv(data, "accent");
  const eyebrow = cv(data, "eyebrow");
  const lead = cv(data, "lead");
  return (
    <section className="hero">
      <div className="container hero-inner">
        <div className="hero-copy">
          {eyebrow ? <span className="pill">{eyebrow}</span> : null}
          <h1 className="hero-title">
            {title}
            {accent ? <span className="grad"> {accent}</span> : null}
          </h1>
          {lead ? <p className="hero-lead">{lead}</p> : null}
          <Ticks points={points} />
          {ctas.length ? (
            <div className="hero-actions">
              {ctas.map((c, i) => (
                <Btn
                  key={i}
                  href={localizeHref(lang, cv(c, "href"))}
                  label={cv(c, "label")}
                  style={cv(c, "style")}
                  big={!!c.big}
                  external={!!c.external}
                />
              ))}
            </div>
          ) : null}
        </div>
        {media ? (
          <div className="hero-media">
            <BrowserWindow media={media} />
          </div>
        ) : null}
      </div>
      {stats.length ? (
        <div className="container">
          <div className="stats">
            {stats.map((s, i) => (
              <div className="s" key={i}>
                <b>{cv(s, "v")}</b>
                <small>{cv(s, "l")}</small>
              </div>
            ))}
          </div>
        </div>
      ) : null}
    </section>
  );
}

function BrowserWindow({ media }: { media: Raw }) {
  const src = cv(media, "src");
  const alt = cv(media, "alt");
  const caption = cv(media, "caption") || "xamltech.com";
  const live = cv(media, "live");
  const width = cv(media, "width") || "1200";
  const height = cv(media, "height") || "700";
  if (!media.window) {
    return <img src={src} alt={alt} loading="eager" />;
  }
  return (
    <figure className="window">
      <div className="window-bar">
        <span className="window-dots" aria-hidden="true">
          <i />
          <i />
          <i />
        </span>
        <span className="window-url">{caption}</span>
        <span className="window-live">
          <b />
          {live}
        </span>
      </div>
      <img src={src} alt={alt} width={width} height={height} loading="eager" />
    </figure>
  );
}

// --- chips ------------------------------------------------------------------

function Chips({ data }: { data: Raw }) {
  const items = arrOf(data.items).map(str);
  if (!items.length) return null;
  return (
    <section className="section section-tight">
      <div className="container chips">
        {items.map((c, i) => (
          <span className="chip chip-accent" key={i}>
            {c}
          </span>
        ))}
      </div>
    </section>
  );
}

// --- features ---------------------------------------------------------------

function Features({ data, lang }: { data: Raw; lang: Lang }) {
  const items = arrOf(data.items);
  const cols = cv(data, "cols") || "3";
  if (!items.length) return null;
  return (
    <section className={`section${data.alt ? " section-alt" : ""}`} id={cv(data, "id") || undefined}>
      <div className="container">
        <SectionHead eyebrow={cv(data, "eyebrow")} title={cv(data, "title")} sub={cv(data, "sub")} />
        <div className={`all-grid cols-${cols}`}>
          {items.map((it, i) => (
            <article className="all-card" key={i}>
              {it.img ? (
                <img
                  className="all-img"
                  src={cv(it, "img")}
                  alt={cv(it, "alt")}
                  width={360}
                  height={200}
                  loading="lazy"
                />
              ) : null}
              <div className="all-head">
                {it.icon ? <div className="all-ico">{str(it.icon)}</div> : null}
                <h3>{cv(it, "title")}</h3>
              </div>
              {it.text ? <p className="lead">{cv(it, "text")}</p> : null}
              {arrOf(it.chips).length ? (
                <div className="all-meta">
                  {arrOf(it.chips).map((c, j) => (
                    <span className="chip" key={j}>
                      {str(c)}
                    </span>
                  ))}
                </div>
              ) : null}
            </article>
          ))}
        </div>
        {data.cta ? (
          <p className="section-cta">
            <Btn href={localizeHref(lang, cv(data.cta as Raw, "href"))} label={cv(data.cta as Raw, "label")} big />
          </p>
        ) : null}
      </div>
    </section>
  );
}

// --- steps ------------------------------------------------------------------

function Steps({ data }: { data: Raw }) {
  const items = arrOf(data.items);
  if (!items.length) return null;
  return (
    <section className={`section${data.alt ? " section-alt" : ""}`}>
      <div className="container">
        <SectionHead eyebrow={cv(data, "eyebrow")} title={cv(data, "title")} sub={cv(data, "sub")} />
        <ol className="steps">
          {items.map((it, i) => (
            <li key={i}>
              <span className="step-num">{i + 1}</span>
              <div>
                <h3>{cv(it, "title")}</h3>
                <p>{cv(it, "text")}</p>
              </div>
            </li>
          ))}
        </ol>
      </div>
    </section>
  );
}

// --- stats ------------------------------------------------------------------

function Stats({ data }: { data: Raw }) {
  const items = arrOf(data.items);
  if (!items.length) return null;
  return (
    <section className="section section-tight">
      <div className="container">
        {data.title ? <SectionHead title={cv(data, "title")} /> : null}
        <div className="stats">
          {items.map((s, i) => (
            <div className="s" key={i}>
              <b>{cv(s, "v")}</b>
              <small>{cv(s, "l")}</small>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}

// --- testimonials -----------------------------------------------------------

function Testimonials({ data }: { data: Raw }) {
  const items = arrOf(data.items);
  if (!items.length) return null;
  return (
    <section className={`section${data.alt ? " section-alt" : ""}`}>
      <div className="container">
        <SectionHead eyebrow={cv(data, "eyebrow")} title={cv(data, "title")} sub={cv(data, "sub")} />
        <div className="quote-grid">
          {items.map((it, i) => (
            <figure className="quote" key={i}>
              {it.stars ? <div className="quote-stars">{str(it.stars)}</div> : null}
              <blockquote>{cv(it, "quote")}</blockquote>
              <figcaption>
                <div className="quote-name">{cv(it, "name")}</div>
                <div className="quote-role">{cv(it, "role")}</div>
              </figcaption>
            </figure>
          ))}
        </div>
      </div>
    </section>
  );
}

// --- faq --------------------------------------------------------------------

function Faq({ data }: { data: Raw }) {
  const items = arrOf(data.items);
  if (!items.length) return null;
  return (
    <section className={`section${data.alt ? " section-alt" : ""}`}>
      <div className="container">
        <SectionHead eyebrow={cv(data, "eyebrow")} title={cv(data, "title")} sub={cv(data, "sub")} />
        <div className="faqs">
          {items.map((it, i) => (
            <details key={i}>
              <summary>{cv(it, "q")}</summary>
              <p>{cv(it, "a")}</p>
            </details>
          ))}
        </div>
      </div>
    </section>
  );
}

// --- split ------------------------------------------------------------------

function Split({ data, lang }: { data: Raw; lang: Lang }) {
  const points = arrOf(data.points).map(str);
  const media = (data.media ?? null) as Raw | null;
  const cta = (data.cta ?? null) as Raw | null;
  return (
    <section className={`section${data.alt ? " section-alt" : ""}`}>
      <div className={`container split${data.reverse ? " split-rev" : ""}`}>
        <div className="split-body">
          <SectionHead eyebrow={cv(data, "eyebrow")} title={cv(data, "title")} sub={cv(data, "lead")} />
          <Ticks points={points} />
          {cta ? (
            <p className="section-cta">
              <Btn href={localizeHref(lang, cv(cta, "href"))} label={cv(cta, "label")} />
            </p>
          ) : null}
        </div>
        {media ? (
          <div className="split-media">
            <img src={cv(media, "src")} alt={cv(media, "alt")} loading="lazy" />
          </div>
        ) : null}
      </div>
    </section>
  );
}

// --- richtext ---------------------------------------------------------------

function RichText({ data }: { data: Raw }) {
  return (
    <section className="section">
      <div className="container prose" id={cv(data, "id") || undefined}>
        <div dangerouslySetInnerHTML={{ __html: cv(data, "html") }} />
      </div>
    </section>
  );
}

// --- pricing ----------------------------------------------------------------

function Pricing({ data, lang }: { data: Raw; lang: Lang }) {
  const items = arrOf(data.items);
  if (!items.length) return null;
  const cols = cv(data, "cols") || "3";
  return (
    <section className={`section${data.alt ? " section-alt" : ""}`} id={cv(data, "id") || undefined}>
      <div className="container">
        <SectionHead eyebrow={cv(data, "eyebrow")} title={cv(data, "title")} sub={cv(data, "sub")} />
        <div className={`plans cols-${cols}`}>
          {items.map((it, i) => {
            const cta = (it.cta ?? null) as Raw | null;
            return (
              <article className={`plan${it.featured ? " featured" : ""}`} key={i}>
                {it.tag ? <span className="plan-tag">{str(it.tag)}</span> : null}
                <h3>{cv(it, "title")}</h3>
                {it.text ? <p className="desc">{cv(it, "text")}</p> : null}
                {it.price ? (
                  <div className="price">
                    {it.currency ? <span className="cur">{str(it.currency)}</span> : null}
                    <span className="amount">{str(it.price)}</span>
                  </div>
                ) : null}
                {arrOf(it.features).length ? <Ticks points={arrOf(it.features).map(str)} /> : null}
                {cta ? (
                  <div className="fill">
                    <Btn href={localizeHref(lang, cv(cta, "href"))} label={cv(cta, "label")} />
                  </div>
                ) : null}
              </article>
            );
          })}
        </div>
      </div>
    </section>
  );
}

// --- roi (interactive) ------------------------------------------------------

function fmtMoney(n: number, currency: string): string {
  return `${currency} ${Math.round(n).toLocaleString("en-US")}`;
}

function Roi({ data }: { data: Raw }) {
  const currency = cv(data, "currency") || "EUR";
  const investment0 = Number(cv(data, "investment")) || 0;
  const monthly0 = Number(cv(data, "monthlyDefault")) || 0;
  const [monthly, setMonthly] = useState(monthly0);
  const [investment, setInvestment] = useState(investment0);
  const m = Number.isFinite(monthly) && monthly >= 0 ? monthly : 0;
  const inv = Number.isFinite(investment) && investment >= 0 ? investment : 0;
  const payback = m > 0 ? `${(inv / m).toFixed(1)} mo` : "—";
  const net = m * 12 - inv;
  const annual = `${net >= 0 ? "+" : ""}${fmtMoney(net, currency)} (${inv > 0 ? Math.round((net / inv) * 100) : 0}%)`;
  return (
    <section className={`section${data.alt ? " section-alt" : ""}`} id="roi">
      <div className="container">
        <SectionHead eyebrow={cv(data, "eyebrow")} title={cv(data, "title")} sub={cv(data, "sub")} />
        <div className="roi">
          <div className="roi-controls">
            <label className="field">
              <span className="field-label">{cv(data, "monthlyLabel")}</span>
              <input
                className="input"
                type="number"
                min={0}
                step={100}
                value={monthly}
                onChange={(e) => setMonthly(Number(e.target.value))}
              />
            </label>
            <label className="field">
              <span className="field-label">{cv(data, "investmentLabel")}</span>
              <input
                className="input"
                type="number"
                min={0}
                step={50}
                value={investment}
                onChange={(e) => setInvestment(Number(e.target.value))}
              />
            </label>
          </div>
          <div className="roi-out">
            <div className="roi-cell">
              <div className="roi-v">{fmtMoney(m, currency)}</div>
              <div className="roi-l">{cv(data, "savingsLabel")}</div>
            </div>
            <div className="roi-cell">
              <div className="roi-v">{payback}</div>
              <div className="roi-l">{cv(data, "paybackLabel")}</div>
            </div>
            <div className="roi-cell">
              <div className="roi-v">{annual}</div>
              <div className="roi-l">{cv(data, "annualLabel")}</div>
            </div>
          </div>
        </div>
      </div>
    </section>
  );
}

// --- articles ---------------------------------------------------------------

function Articles({ data, lang }: { data: Raw; lang: Lang }) {
  const items = arrOf(data.items);
  const more = cv(data, "more");
  if (!items.length) return null;
  return (
    <section className={`section${data.alt ? " section-alt" : ""}`}>
      <div className="container">
        <SectionHead eyebrow={cv(data, "eyebrow")} title={cv(data, "title")} sub={cv(data, "sub")} />
        <div className="all-grid cols-3">
          {items.map((it, i) => (
            <a className="all-card link" href={localizeHref(lang, cv(it, "href"))} key={i}>
              {it.tag ? <span className="chip chip-accent">{str(it.tag)}</span> : null}
              <h3>{cv(it, "title")}</h3>
              <p className="lead">{cv(it, "text")}</p>
              <span className="card-more">{more}</span>
            </a>
          ))}
        </div>
      </div>
    </section>
  );
}

// --- cta band ---------------------------------------------------------------

function Cta({ data, lang }: { data: Raw; lang: Lang }) {
  const actions = arrOf(data.actions);
  return (
    <section className="section">
      <div className="container">
        <div className="cta-band">
          <h2>{cv(data, "title")}</h2>
          {data.text ? <p>{str(data.text)}</p> : null}
          {actions.length ? (
            <div className="cta-actions">
              {actions.map((a, i) => (
                <Btn
                  key={i}
                  href={localizeHref(lang, cv(a, "href"))}
                  label={cv(a, "label")}
                  style={cv(a, "style")}
                  big
                />
              ))}
            </div>
          ) : null}
        </div>
      </div>
    </section>
  );
}

// --- logos ------------------------------------------------------------------

function Logos({ data }: { data: Raw }) {
  const items = arrOf(data.items);
  if (!items.length) return null;
  return (
    <section className="section section-tight">
      <div className="container">
        {data.title ? <p className="section-sub center">{str(data.title)}</p> : null}
        <div className="logo-row">
          {items.map((it, i) => (
            <img key={i} src={cv(it, "src")} alt={cv(it, "alt")} loading="lazy" />
          ))}
        </div>
      </div>
    </section>
  );
}

// --- dispatcher -------------------------------------------------------------

export function Section({ partial, data, lang }: { partial: string; data: Raw; lang: Lang }): ReactNode {
  switch (partial) {
    case "hero":
      return <Hero data={data} lang={lang} />;
    case "chips":
      return <Chips data={data} />;
    case "features":
      return <Features data={data} lang={lang} />;
    case "logos":
      return <Logos data={data} />;
    case "steps":
      return <Steps data={data} />;
    case "stats":
      return <Stats data={data} />;
    case "testimonials":
      return <Testimonials data={data} />;
    case "faq":
      return <Faq data={data} />;
    case "split":
      return <Split data={data} lang={lang} />;
    case "richtext":
      return <RichText data={data} />;
    case "pricing":
      return <Pricing data={data} lang={lang} />;
    case "roi":
      return <Roi data={data} />;
    case "articles":
      return <Articles data={data} lang={lang} />;
    case "cta":
      return <Cta data={data} lang={lang} />;
    default:
      return null;
  }
}

export { SectionHead, Btn };