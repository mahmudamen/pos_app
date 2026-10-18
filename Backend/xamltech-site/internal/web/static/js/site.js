/* XamlTech website — interactions */
(function () {
  "use strict";

  // Theme toggle (persisted; applied pre-paint from localStorage in <head>).
  var root = document.documentElement;
  var themeBtns = document.querySelectorAll("[data-theme-toggle]");
  function syncThemeLabel() {
    var light = root.dataset.theme === "light";
    themeBtns.forEach(function (b) {
      b.setAttribute("aria-label", light ? "Switch to dark mode" : "Switch to light mode");
    });
  }
  themeBtns.forEach(function (btn) {
    btn.addEventListener("click", function () {
      var next = root.dataset.theme === "light" ? "dark" : "light";
      root.dataset.theme = next;
      var tc = document.querySelector('meta[name="theme-color"]');
      if (tc) tc.setAttribute("content", next === "dark" ? "#080b13" : "#ffffff");
      try { localStorage.setItem("xt-theme", next); } catch (e) {}
      syncThemeLabel();
    });
  });
  syncThemeLabel();

  // Mobile menu.
  var burger = document.querySelector("[data-menu-toggle]");
  var scrim = document.querySelector("[data-menu-close]");
  function closeMenu() {
    document.body.classList.remove("menu-open");
    if (scrim) scrim.hidden = true;
    if (burger) burger.setAttribute("aria-expanded", "false");
  }
  if (burger) {
    burger.addEventListener("click", function () {
      var open = document.body.classList.toggle("menu-open");
      if (scrim) scrim.hidden = !open;
      burger.setAttribute("aria-expanded", String(open));
    });
  }
  if (scrim) scrim.addEventListener("click", closeMenu);

  // Mobile submenu accordion.
  document.querySelectorAll(".has-menu > .nav__link").forEach(function (link) {
    link.addEventListener("click", function (e) {
      if (window.matchMedia("(max-width: 900px)").matches) {
        e.preventDefault();
        var item = link.parentElement;
        var open = item.classList.toggle("open");
        link.setAttribute("aria-expanded", String(open));
      }
    });
  });

  // Reveal on scroll.
  var revealables = document.querySelectorAll(".reveal");
  if ("IntersectionObserver" in window && revealables.length) {
    var io = new IntersectionObserver(function (entries) {
      entries.forEach(function (en) {
        if (en.isIntersecting) {
          en.target.classList.add("in");
          io.unobserve(en.target);
        }
      });
    }, { rootMargin: "0px 0px -8% 0px", threshold: 0.06 });
    revealables.forEach(function (el) { io.observe(el); });
  } else {
    revealables.forEach(function (el) { el.classList.add("in"); });
  }

  // Close the mobile menu after navigating.
  document.querySelectorAll(".nav a").forEach(function (a) {
    a.addEventListener("click", function () {
      if (window.matchMedia("(max-width: 900px)").matches) closeMenu();
    });
  });

  // ROI calculator.
  document.querySelectorAll("[data-roi]").forEach(function (box) {
    var monthly = box.querySelector("[data-roi-monthly]");
    var invest = box.querySelector("[data-roi-investment]");
    var savings = box.querySelector("[data-roi-savings]");
    var payback = box.querySelector("[data-roi-payback]");
    var annual = box.querySelector("[data-roi-annual]");
    var currency = box.getAttribute("data-currency") || "EUR";
    function num(el) { var n = parseFloat(el && el.value); return isFinite(n) && n >= 0 ? n : 0; }
    function fmt(n) {
      return currency + " " + Math.round(n).toLocaleString("en-US");
    }
    function calc() {
      var m = num(monthly);
      var inv = num(invest);
      if (savings) savings.textContent = fmt(m);
      if (payback) payback.textContent = m > 0 ? (inv / m).toFixed(1) + " mo" : "—";
      var net = m * 12 - inv;
      if (annual) annual.textContent = (net >= 0 ? "+" : "") + fmt(net) + " (" + (inv > 0 ? Math.round((net / inv) * 100) : 0) + "%)";
    }
    if (monthly) monthly.addEventListener("input", calc);
    if (invest) invest.addEventListener("input", calc);
    calc();
  });
})();
