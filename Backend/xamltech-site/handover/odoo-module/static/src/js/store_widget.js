/**
 * XamlTech Addons Store + Theme Toggle + Language Fix
 * Vanilla JS — no Odoo module system dependency
 */
(function () {
  "use strict";

  // ═══════════════════════════════════════════════
  // THEME TOGGLE (Dark / Light)
  // ═══════════════════════════════════════════════
  function initTheme() {
    var saved = localStorage.getItem("xt-theme");
    var prefer = window.matchMedia("(prefers-color-scheme: light)").matches
      ? "light"
      : "dark";
    var theme = saved || prefer;
    applyTheme(theme);

    // Listen for system changes
    window
      .matchMedia("(prefers-color-scheme: light)")
      .addEventListener("change", function (e) {
        if (!localStorage.getItem("xt-theme")) {
          applyTheme(e.matches ? "light" : "dark");
        }
      });
  }

  function applyTheme(theme) {
    document.documentElement.setAttribute("data-xt-theme", theme);
    document.body.classList.toggle("xt-light", theme === "light");
    var btn = document.getElementById("xt-theme-btn");
    if (btn) btn.textContent = theme === "dark" ? "☀️" : "🌙";
  }

  function toggleTheme() {
    var current =
      document.documentElement.getAttribute("data-xt-theme") || "dark";
    var next = current === "dark" ? "light" : "dark";
    localStorage.setItem("xt-theme", next);
    applyTheme(next);
  }

  // ═══════════════════════════════════════════════
  // LANGUAGE SWITCH FIX
  // Clear SW cache on language change to prevent stale content
  // ═══════════════════════════════════════════════
  function fixLanguageSwitch() {
    var langLinks = document.querySelectorAll(
      ".js_language_selector a[data-url_code], a[hreflang]",
    );
    langLinks.forEach(function (link) {
      link.addEventListener("click", function () {
        // Clear all SW caches before language switch
        if ("caches" in window) {
          caches.keys().then(function (names) {
            names.forEach(function (name) {
              caches.delete(name);
            });
          });
        }
        // Unregister SW temporarily so fresh content loads
        if ("serviceWorker" in navigator) {
          navigator.serviceWorker.getRegistrations().then(function (regs) {
            regs.forEach(function (r) {
              r.unregister();
            });
          });
        }
      });
    });
  }

  // ═══════════════════════════════════════════════
  // DEMO FORM (homepage + /request-demo page)
  // IDs: demo-submit, demo-name, demo-wa, demo-interest, demo-status, demo-modal, modal-close
  // ═══════════════════════════════════════════════
  function initDemoForm() {
    var m = document.getElementById("demo-modal");
    var b = document.getElementById("demo-submit");
    if (!b) return;

    var origText = b.textContent;

    b.addEventListener("click", function () {
      var n = document.getElementById("demo-name").value.trim();
      var w = document.getElementById("demo-wa").value.trim();
      var intEl = document.getElementById("demo-interest");
      var i = intEl ? intEl.value : "";
      var s = document.getElementById("demo-status");
      if (!n || !w) {
        if (s) {
          s.textContent = "أدخل الاسم والواتساب";
          s.style.display = "block";
          s.style.color = "#E8364F";
          s.style.background = "rgba(232,54,79,0.1)";
        }
        return;
      }
      b.textContent = "⏳...";
      b.disabled = true;
      fetch("/request-demo/submit", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          jsonrpc: "2.0",
          method: "call",
          params: { name: n, whatsapp: w, addon_interest: i },
        }),
      })
        .then(function (r) {
          return r.json();
        })
        .then(function () {
          if (m) m.style.display = "flex";
        })
        .catch(function () {
          if (m) m.style.display = "flex";
        })
        .finally(function () {
          b.textContent = origText;
          b.disabled = false;
        });
    });

    var closeBtn = document.getElementById("modal-close");
    if (closeBtn)
      closeBtn.addEventListener("click", function () {
        if (m) m.style.display = "none";
      });
    var bg = m ? m.querySelector(".xt-modal__bg") : null;
    if (bg)
      bg.addEventListener("click", function () {
        if (m) m.style.display = "none";
      });
  }

  // ═══════════════════════════════════════════════
  // CONTACT FORM (/contact-us page)
  // IDs: ct-send, ct-wa, ct-name, ct-email, ct-phone, ct-subject, ct-msg, ct-status
  // ═══════════════════════════════════════════════
  function initContactForm() {
    var waBtn = document.getElementById("ct-wa");
    var sendBtn = document.getElementById("ct-send");
    if (!waBtn && !sendBtn) return;

    var W = "201007982468";

    if (waBtn) {
      waBtn.addEventListener("click", function () {
        var n = document.getElementById("ct-name").value.trim();
        var e = document.getElementById("ct-email").value.trim();
        var msg = document.getElementById("ct-msg").value.trim();
        var st = document.getElementById("ct-status");
        if (!n || !e || !msg) {
          if (st) {
            st.textContent = "Fill Name, Email, Message.";
            st.style.display = "block";
          }
          return;
        }
        var t =
          "🔔 *XamlTech Inquiry*\n\n👤 *Name:* " +
          n +
          "\n📧 *Email:* " +
          e +
          "\n💬 *Message:*\n" +
          msg;
        window.open(
          "https://wa.me/" + W + "?text=" + encodeURIComponent(t),
          "_blank",
        );
      });
    }

    if (sendBtn) {
      sendBtn.addEventListener("click", function () {
        var n = document.getElementById("ct-name").value.trim();
        var e = document.getElementById("ct-email").value.trim();
        var p = document.getElementById("ct-phone").value.trim();
        var s = document.getElementById("ct-subject").value;
        var msg = document.getElementById("ct-msg").value.trim();
        var st = document.getElementById("ct-status");
        if (!n || !e || !msg) {
          if (st) {
            st.textContent = "Fill Name, Email, Message.";
            st.style.display = "block";
          }
          return;
        }
        sendBtn.textContent = "Sending...";
        sendBtn.disabled = true;
        fetch("/contact-us/send", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({
            jsonrpc: "2.0",
            method: "call",
            params: { name: n, email: e, phone: p, subject: s, message: msg },
          }),
        })
          .then(function (r) {
            return r.json();
          })
          .then(function () {
            if (st) {
              st.textContent = "✅ Sent!";
              st.style.display = "block";
              st.style.color = "#10B981";
            }
          })
          .catch(function () {
            if (st) {
              st.textContent = "Failed. Try WhatsApp.";
              st.style.display = "block";
            }
          })
          .finally(function () {
            sendBtn.textContent = "Send Message";
            sendBtn.disabled = false;
          });
      });
    }
  }

  // ═══════════════════════════════════════════════
  // STORE: Search, Filter, Sort, Animations
  // ═══════════════════════════════════════════════
  function initStore() {
    var store = document.getElementById("rs-addons-store");
    if (!store) return;

    var cards = store.querySelectorAll(".xt-acard[data-addon-id]");
    var grid = store.querySelector(".xt-grid3");
    var searchInput = document.getElementById("rs-search-input");
    var sortSelect = document.getElementById("rs-sort-select");
    var countEl = document.getElementById("rs-addon-count");
    var catBtns = store.querySelectorAll(".xt-cat");

    var currentCat = "All";
    var currentQ = "";
    var currentSort = "popular";
    var searchTimeout = null;

    if (searchInput) {
      searchInput.addEventListener("input", function () {
        clearTimeout(searchTimeout);
        searchTimeout = setTimeout(function () {
          currentQ = (searchInput.value || "").trim().toLowerCase();
          filterAndSort();
        }, 300);
      });
    }

    if (sortSelect) {
      sortSelect.addEventListener("change", function () {
        currentSort = sortSelect.value;
        filterAndSort();
      });
    }

    catBtns.forEach(function (btn) {
      btn.addEventListener("click", function () {
        currentCat = btn.dataset.category || "All";
        catBtns.forEach(function (b) {
          b.classList.toggle("xt-cat--on", b.dataset.category === currentCat);
        });
        filterAndSort();
      });
    });

    cards.forEach(function (card) {
      card.style.cursor = "pointer";
      card.addEventListener("click", function () {
        var url = card.dataset.storeUrl;
        var slug = card.dataset.slug;
        if (url) window.open(url, "_blank");
        else if (slug) window.location.href = "/addons-store/" + slug;
      });
    });

    function filterAndSort() {
      var count = 0;
      cards.forEach(function (card) {
        var cat = card.dataset.category || "";
        var name = (card.dataset.name || "").toLowerCase();
        var tags = (card.dataset.tags || "").toLowerCase();
        var desc = (card.dataset.description || "").toLowerCase();
        var show = currentCat === "All" || cat === currentCat;
        if (show && currentQ) {
          show =
            name.indexOf(currentQ) !== -1 ||
            tags.indexOf(currentQ) !== -1 ||
            desc.indexOf(currentQ) !== -1;
        }
        card.style.display = show ? "" : "none";
        if (show) count++;
      });

      if (grid) {
        var arr = Array.prototype.slice.call(cards);
        var fns = {
          popular: function (a, b) {
            return (
              (parseInt(b.dataset.downloads) || 0) -
              (parseInt(a.dataset.downloads) || 0)
            );
          },
          rating: function (a, b) {
            return (
              (parseFloat(b.dataset.rating) || 0) -
              (parseFloat(a.dataset.rating) || 0)
            );
          },
          newest: function (a, b) {
            return (
              (parseInt(b.dataset.addonId) || 0) -
              (parseInt(a.dataset.addonId) || 0)
            );
          },
          price_low: function (a, b) {
            return (
              (parseFloat(a.dataset.price) || 0) -
              (parseFloat(b.dataset.price) || 0)
            );
          },
          price_high: function (a, b) {
            return (
              (parseFloat(b.dataset.price) || 0) -
              (parseFloat(a.dataset.price) || 0)
            );
          },
        };
        if (fns[currentSort]) {
          arr.sort(fns[currentSort]);
          arr.forEach(function (c) {
            grid.appendChild(c);
          });
        }
      }

      if (countEl)
        countEl.textContent = count + " addon" + (count !== 1 ? "s" : "");
      animateCards();
    }

    function animateCards() {
      var idx = 0;
      cards.forEach(function (card) {
        if (card.style.display === "none") {
          return;
        }
        card.style.transitionDelay = idx * 60 + "ms";
        card.style.opacity = "0";
        card.style.transform = "translateY(20px)";
        card.style.transition = "opacity 0.4s ease, transform 0.4s ease";
        idx++;
      });
      requestAnimationFrame(function () {
        requestAnimationFrame(function () {
          cards.forEach(function (card) {
            if (card.style.display === "none") {
              return;
            }
            card.style.opacity = "1";
            card.style.transform = "translateY(0)";
          });
        });
      });
    }

    animateCards();
  }

  // ═══════════════════════════════════════════════
  // SCROLL REVEAL (intersection observer + stagger)
  // ═══════════════════════════════════════════════
  function initScrollReveal() {
    if (!("IntersectionObserver" in window)) return;
    var targets = document.querySelectorAll(
      ".xt-card, .xt-stat, .xt-pillar, .xt-why, .xt-step, .xt-ind, .xt-partner, .xt-info, .xt-hcard, .xt-acard, .xt-pill",
    );
    if (!targets.length) return;
    var observer = new IntersectionObserver(
      function (entries) {
        entries.forEach(function (entry) {
          if (entry.isIntersecting) {
            entry.target.classList.add("xt-sr-visible");
            observer.unobserve(entry.target);
          }
        });
      },
      { threshold: 0.1, rootMargin: "0px 0px -40px 0px" },
    );

    requestAnimationFrame(function () {
      var idx = 0;
      targets.forEach(function (el) {
        el.classList.add("xt-sr-hidden");
        el.style.animationDelay = (idx % 6) * 80 + "ms";
        observer.observe(el);
        idx++;
      });
    });
  }

  // ═══════════════════════════════════════════════
  // 3D CARD TILT (mousemove on data-xt-tilt elements)
  // ═══════════════════════════════════════════════
  function init3DTilt() {
    var cards = document.querySelectorAll("[data-xt-tilt]");
    cards.forEach(function (card) {
      card.addEventListener("mousemove", function (e) {
        var rect = card.getBoundingClientRect();
        var x = (e.clientX - rect.left) / rect.width - 0.5;
        var y = (e.clientY - rect.top) / rect.height - 0.5;
        card.style.transform =
          "perspective(800px) rotateY(" +
          x * 8 +
          "deg) rotateX(" +
          -y * 8 +
          "deg) translateZ(10px)";
        card.style.transition = "transform 0.1s ease";
      });
      card.addEventListener("mouseleave", function () {
        card.style.transform = "";
        card.style.transition = "transform 0.5s cubic-bezier(0.4,0,0.2,1)";
      });
    });
  }

  // ═══════════════════════════════════════════════
  // COUNT-UP ANIMATION (stats numbers)
  // ═══════════════════════════════════════════════
  function animateCount(el, start, end, suffix, duration) {
    var startTime = null;
    function step(timestamp) {
      if (!startTime) startTime = timestamp;
      var progress = Math.min((timestamp - startTime) / duration, 1);
      var eased = 1 - Math.pow(1 - progress, 3);
      el.textContent = Math.floor(eased * (end - start) + start) + suffix;
      if (progress < 1) requestAnimationFrame(step);
    }
    requestAnimationFrame(step);
  }

  function initCountUp() {
    var stats = document.querySelectorAll("[data-count-target]");
    if (!stats.length || !("IntersectionObserver" in window)) return;
    var observer = new IntersectionObserver(
      function (entries) {
        entries.forEach(function (entry) {
          if (!entry.isIntersecting) return;
          var el = entry.target;
          var target = parseInt(el.dataset.countTarget) || 0;
          var suffix = el.dataset.countSuffix || "";
          animateCount(el, 0, target, suffix, 1500);
          observer.unobserve(el);
        });
      },
      { threshold: 0.5 },
    );
    stats.forEach(function (s) {
      observer.observe(s);
    });
  }

  // ═══════════════════════════════════════════════
  // INJECT THEME TOGGLE BUTTON
  // ═══════════════════════════════════════════════
  function injectThemeToggle() {
    if (document.getElementById("xt-theme-btn")) return;
    var btn = document.createElement("button");
    btn.id = "xt-theme-btn";
    btn.className = "xt-theme-toggle";
    btn.setAttribute("aria-label", "Toggle dark/light mode");
    btn.textContent = "☀️";
    btn.addEventListener("click", toggleTheme);
    document.body.appendChild(btn);
  }

  // ═══════════════════════════════════════════════
  // INIT
  // ═══════════════════════════════════════════════
  document.addEventListener("DOMContentLoaded", function () {
    initTheme();
    injectThemeToggle();
    fixLanguageSwitch();
    initDemoForm();
    initContactForm();
    initStore();
    initScrollReveal();
    init3DTilt();
    initCountUp();
  });
})();
