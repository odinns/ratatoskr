const prefersReducedMotion = window.matchMedia('(prefers-reduced-motion: reduce)');

const revealItems = document.querySelectorAll('[data-reveal]');
const root = document.documentElement;

if (!prefersReducedMotion.matches) {
  const observer = new IntersectionObserver(
    (entries) => {
      entries.forEach((entry) => {
        if (entry.isIntersecting) {
          entry.target.classList.add('is-visible');
          observer.unobserve(entry.target);
        }
      });
    },
    { rootMargin: '0px 0px -8% 0px' },
  );

  revealItems.forEach((item, index) => {
    item.style.setProperty('--reveal-delay', `${Math.min(index * 45, 220)}ms`);
    observer.observe(item);
  });
} else {
  revealItems.forEach((item) => item.classList.add('is-visible'));
}

const scanLog = document.querySelector('[data-scan-log]');
const lines = [
  '$ ratatoskr report --path ~/Code',
  'need: 17 GB for dataset processing',
  'safe     7.4 GB  logs, build output, caches',
  'cautious 15.9 GB  dependencies, package stores',
  'report   1.5 GB  unknown large files',
  'next: ask ratatoskr-report-analysis',
];

if (scanLog && !prefersReducedMotion.matches) {
  let lineIndex = 0;
  let charIndex = 0;

  const typeNext = () => {
    const visibleLines = lines.slice(0, lineIndex);
    const currentLine = lines[lineIndex] || '';

    scanLog.textContent = [...visibleLines, currentLine.slice(0, charIndex)].join('\n');
    charIndex += 1;

    if (charIndex > currentLine.length) {
      lineIndex += 1;
      charIndex = 0;
    }

    if (lineIndex >= lines.length) {
      window.setTimeout(() => {
        lineIndex = 0;
        charIndex = 0;
        typeNext();
      }, 2200);
      return;
    }

    window.setTimeout(typeNext, charIndex === 1 ? 240 : 20);
  };

  typeNext();
}

if (!prefersReducedMotion.matches) {
  const heroStage = document.querySelector('.hero-stage');
  let pendingFrame = 0;

  const setMotionVars = (clientX, clientY) => {
    if (pendingFrame) {
      window.cancelAnimationFrame(pendingFrame);
    }

    pendingFrame = window.requestAnimationFrame(() => {
      const centerX = window.innerWidth / 2;
      const centerY = window.innerHeight / 2;
      const offsetX = (clientX - centerX) / centerX;
      const offsetY = (clientY - centerY) / centerY;

      root.style.setProperty('--cursor-x', `${Math.round(offsetX * 18)}px`);
      root.style.setProperty('--cursor-y', `${Math.round(offsetY * 14)}px`);

      if (heroStage) {
        heroStage.style.setProperty('--hero-x', (offsetX * 8).toFixed(2));
        heroStage.style.setProperty('--hero-y', (offsetY * 6).toFixed(2));
      }
    });
  };

  window.addEventListener('pointermove', (event) => {
    setMotionVars(event.clientX, event.clientY);
  }, { passive: true });

  window.addEventListener('pointerleave', () => {
    root.style.removeProperty('--cursor-x');
    root.style.removeProperty('--cursor-y');

    if (heroStage) {
      heroStage.style.removeProperty('--hero-x');
      heroStage.style.removeProperty('--hero-y');
    }
  });
}
