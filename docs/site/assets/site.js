const prefersReducedMotion = window.matchMedia('(prefers-reduced-motion: reduce)');

const revealItems = document.querySelectorAll('[data-reveal]');

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
