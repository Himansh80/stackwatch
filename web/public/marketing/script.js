/* ===========================================================
 * StackWatch Marketing Site — script.js
 * Tier 12 Phase 2 · Pure vanilla JS · No frameworks · ~50 LOC
 * Three behaviors: FAQ accordion, KPI counter, smooth scroll.
 * =========================================================== */

(function () {
  'use strict';

  // -------- 1. FAQ accordion --------
  // Click question → toggle .open on parent .faq-item
  // CSS handles the animation (max-height transition).
  function initFAQ() {
    var questions = document.querySelectorAll('.faq-question');
    for (var i = 0; i < questions.length; i++) {
      questions[i].addEventListener('click', function () {
        var item = this.closest('.faq-item');
        if (!item) return;
        var wasOpen = item.classList.contains('open');
        // Close all other items (accordion behavior)
        var items = document.querySelectorAll('.faq-item.open');
        for (var j = 0; j < items.length; j++) {
          items[j].classList.remove('open');
          var btn = items[j].querySelector('.faq-question');
          if (btn) btn.setAttribute('aria-expanded', 'false');
        }
        if (!wasOpen) {
          item.classList.add('open');
          this.setAttribute('aria-expanded', 'true');
        }
      });
    }
  }

  // -------- 2. KPI counter (IntersectionObserver) --------
  // When .kpi-value enters viewport, animate from 0 to data-target.
  // Smooth easing, ~1.6s duration, fires once.
  function animateCounter(el, target, duration) {
    var start = 0;
    var startTime = null;
    function step(timestamp) {
      if (!startTime) startTime = timestamp;
      var progress = Math.min((timestamp - startTime) / duration, 1);
      // Ease-out cubic
      var eased = 1 - Math.pow(1 - progress, 3);
      var current = Math.floor(eased * target);
      el.textContent = current;
      if (progress < 1) {
        window.requestAnimationFrame(step);
      } else {
        el.textContent = target;
      }
    }
    window.requestAnimationFrame(step);
  }

  function initKPICounters() {
    var kpis = document.querySelectorAll('.kpi-value[data-target]');
    if (!kpis.length) return;
    if (!('IntersectionObserver' in window)) {
      // Fallback: just set the final value
      for (var i = 0; i < kpis.length; i++) {
        kpis[i].textContent = kpis[i].getAttribute('data-target');
      }
      return;
    }
    var observer = new IntersectionObserver(function (entries) {
      for (var i = 0; i < entries.length; i++) {
        var entry = entries[i];
        if (entry.isIntersecting) {
          var el = entry.target;
          var target = parseInt(el.getAttribute('data-target'), 10) || 0;
          animateCounter(el, target, 1600);
          observer.unobserve(el);
        }
      }
    }, { threshold: 0.4 });
    for (var j = 0; j < kpis.length; j++) {
      observer.observe(kpis[j]);
    }
  }

  // -------- 3. Smooth scroll for # anchors --------
  // CSS handles most browsers via `scroll-behavior: smooth`,
  // but Safari needs a polyfill that respects `prefers-reduced-motion`.
  function initSmoothScroll() {
    var prefersReduced = window.matchMedia &&
      window.matchMedia('(prefers-reduced-motion: reduce)').matches;
    var links = document.querySelectorAll('a[href^="#"]');
    for (var i = 0; i < links.length; i++) {
      links[i].addEventListener('click', function (e) {
        var href = this.getAttribute('href');
        if (!href || href === '#') return;
        var target = document.querySelector(href);
        if (!target) return;
        e.preventDefault();
        target.scrollIntoView({
          behavior: prefersReduced ? 'auto' : 'smooth',
          block: 'start'
        });
      });
    }
  }

  // -------- Bootstrap --------
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', function () {
      initFAQ();
      initKPICounters();
      initSmoothScroll();
    });
  } else {
    initFAQ();
    initKPICounters();
    initSmoothScroll();
  }
})();