/**
 * XamlTech Blog — Scroll Reveal + FAQ Accordion + Particle Canvas + Counter Animation
 * Vanilla JS — no framework dependency
 */
(function () {
  "use strict";

  // ═══════════════════════════════════════════════
  // SCROLL REVEAL (IntersectionObserver)
  // ═══════════════════════════════════════════════
  function initScrollReveal() {
    var prefersReduced = window.matchMedia(
      "(prefers-reduced-motion: reduce)",
    ).matches;
    if (prefersReduced) {
      document
        .querySelectorAll(
          ".xt-reveal, .xt-reveal--scale, .xt-reveal--left, .xt-reveal--right",
        )
        .forEach(function (el) {
          el.classList.add("xt-revealed");
        });
      return;
    }

    var observer = new IntersectionObserver(
      function (entries) {
        entries.forEach(function (entry) {
          if (entry.isIntersecting) {
            entry.target.classList.add("xt-revealed");
            observer.unobserve(entry.target);
          }
        });
      },
      { threshold: 0.12, rootMargin: "0px 0px -40px 0px" },
    );

    document
      .querySelectorAll(
        ".xt-reveal, .xt-reveal--scale, .xt-reveal--left, .xt-reveal--right",
      )
      .forEach(function (el) {
        observer.observe(el);
      });
  }

  // ═══════════════════════════════════════════════
  // FAQ ACCORDION (smooth grid-template-rows)
  // ═══════════════════════════════════════════════
  function initFaqAccordion() {
    document.querySelectorAll(".xt-blog-faq__question").forEach(function (btn) {
      btn.addEventListener("click", function () {
        var item = btn.closest(".xt-blog-faq__item");
        var wasOpen = item.classList.contains("xt-faq-open");

        item
          .closest(".xt-blog-faq")
          .querySelectorAll(".xt-blog-faq__item")
          .forEach(function (sibling) {
            sibling.classList.remove("xt-faq-open");
          });

        if (!wasOpen) {
          item.classList.add("xt-faq-open");
        }
      });
    });
  }

  // ═══════════════════════════════════════════════
  // ANIMATED COUNTERS (count up on scroll)
  // ═══════════════════════════════════════════════
  function initCounters() {
    var prefersReduced = window.matchMedia(
      "(prefers-reduced-motion: reduce)",
    ).matches;

    var observer = new IntersectionObserver(
      function (entries) {
        entries.forEach(function (entry) {
          if (entry.isIntersecting) {
            animateCounter(entry.target, prefersReduced);
            observer.unobserve(entry.target);
          }
        });
      },
      { threshold: 0.5 },
    );

    document.querySelectorAll("[data-count-to]").forEach(function (el) {
      observer.observe(el);
    });
  }

  function animateCounter(el, instant) {
    var target = parseInt(el.getAttribute("data-count-to"), 10);
    var suffix = el.getAttribute("data-count-suffix") || "";
    var prefix = el.getAttribute("data-count-prefix") || "";
    var duration = 1600;

    if (instant || isNaN(target)) {
      el.textContent = prefix + target + suffix;
      return;
    }

    var start = 0;
    var startTime = null;

    function step(timestamp) {
      if (!startTime) startTime = timestamp;
      var progress = Math.min((timestamp - startTime) / duration, 1);
      var eased = 1 - Math.pow(1 - progress, 4);
      var current = Math.round(start + (target - start) * eased);
      el.textContent = prefix + current + suffix;
      if (progress < 1) {
        requestAnimationFrame(step);
      }
    }

    requestAnimationFrame(step);
  }

  // ═══════════════════════════════════════════════
  // PARTICLE CANVAS (lightweight dots + connections)
  // ═══════════════════════════════════════════════
  function initParticles() {
    var canvas = document.querySelector(".xt-particles-canvas");
    if (!canvas) return;

    var prefersReduced = window.matchMedia(
      "(prefers-reduced-motion: reduce)",
    ).matches;
    if (prefersReduced) return;

    var ctx = canvas.getContext("2d");
    var particles = [];
    var particleCount = 40;
    var connectDistance = 120;
    var animId;

    function resize() {
      var rect = canvas.parentElement.getBoundingClientRect();
      canvas.width = rect.width;
      canvas.height = rect.height;
    }

    function createParticles() {
      particles = [];
      for (var i = 0; i < particleCount; i++) {
        particles.push({
          x: Math.random() * canvas.width,
          y: Math.random() * canvas.height,
          vx: (Math.random() - 0.5) * 0.4,
          vy: (Math.random() - 0.5) * 0.4,
          size: Math.random() * 1.5 + 0.5,
          opacity: Math.random() * 0.3 + 0.1,
        });
      }
    }

    function draw() {
      ctx.clearRect(0, 0, canvas.width, canvas.height);

      for (var i = 0; i < particles.length; i++) {
        var p = particles[i];
        p.x += p.vx;
        p.y += p.vy;

        if (p.x < 0 || p.x > canvas.width) p.vx *= -1;
        if (p.y < 0 || p.y > canvas.height) p.vy *= -1;

        ctx.beginPath();
        ctx.arc(p.x, p.y, p.size, 0, Math.PI * 2);
        ctx.fillStyle = "rgba(0, 173, 238, " + p.opacity + ")";
        ctx.fill();

        for (var j = i + 1; j < particles.length; j++) {
          var p2 = particles[j];
          var dx = p.x - p2.x;
          var dy = p.y - p2.y;
          var dist = Math.sqrt(dx * dx + dy * dy);
          if (dist < connectDistance) {
            var alpha = (1 - dist / connectDistance) * 0.08;
            ctx.beginPath();
            ctx.moveTo(p.x, p.y);
            ctx.lineTo(p2.x, p2.y);
            ctx.strokeStyle = "rgba(0, 173, 238, " + alpha + ")";
            ctx.lineWidth = 0.5;
            ctx.stroke();
          }
        }
      }

      animId = requestAnimationFrame(draw);
    }

    resize();
    createParticles();
    draw();

    var resizeTimeout;
    window.addEventListener("resize", function () {
      clearTimeout(resizeTimeout);
      resizeTimeout = setTimeout(function () {
        resize();
        createParticles();
      }, 200);
    });

    var heroObserver = new IntersectionObserver(
      function (entries) {
        if (entries[0].isIntersecting) {
          if (!animId) draw();
        } else {
          cancelAnimationFrame(animId);
          animId = null;
        }
      },
      { threshold: 0 },
    );
    heroObserver.observe(canvas.parentElement);
  }

  // ═══════════════════════════════════════════════
  // MAGNETIC HOVER (subtle card tilt on mousemove)
  // ═══════════════════════════════════════════════
  function initMagneticCards() {
    var prefersReduced = window.matchMedia(
      "(prefers-reduced-motion: reduce)",
    ).matches;
    if (prefersReduced) return;

    document
      .querySelectorAll(".xt-blog-featured__card")
      .forEach(function (card) {
        card.addEventListener("mousemove", function (e) {
          var rect = card.getBoundingClientRect();
          var x = (e.clientX - rect.left) / rect.width - 0.5;
          var y = (e.clientY - rect.top) / rect.height - 0.5;
          card.style.transform =
            "perspective(800px) rotateY(" +
            x * 4 +
            "deg) rotateX(" +
            -y * 4 +
            "deg) translateY(-6px)";
        });

        card.addEventListener("mouseleave", function () {
          card.style.transform = "";
        });
      });
  }

  // ═══════════════════════════════════════════════
  // INIT ON DOM READY
  // ═══════════════════════════════════════════════
  function init() {
    initScrollReveal();
    initFaqAccordion();
    initCounters();
    initParticles();
    initMagneticCards();
  }

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", init);
  } else {
    init();
  }
})();
