/* ozkansen.com — paylaşılan Alpine.js bileşeni.
   Tüm sayfalarda ortak davranışlar: dark mode, scroll durumu,
   aktif bölüm takibi, mobil menü ve scroll-reveal animasyonu. */
function site() {
  return {
    dark: false,
    mm: false,
    sc: false,
    s: 'hero',

    init() {
      // Dark mode: localStorage tercihi, yoksa sistem tercihi
      this.dark =
        localStorage.getItem('theme') === 'dark' ||
        (!localStorage.getItem('theme') &&
          window.matchMedia('(prefers-color-scheme: dark)').matches);

      this.$watch('dark', (v) =>
        localStorage.setItem('theme', v ? 'dark' : 'light')
      );

      // Scroll: header durumu + aktif bölüm takibi
      window.addEventListener(
        'scroll',
        () => {
          this.sc = window.scrollY > 20;
          this.updateSection();
        },
        { passive: true }
      );

      // Scroll-reveal animasyonu
      const io = new IntersectionObserver(
        (entries) => {
          entries.forEach((e) => {
            if (e.isIntersecting) {
              e.target.classList.add('in');
              io.unobserve(e.target);
            }
          });
        },
        { threshold: 0.1, rootMargin: '0px 0px -40px 0px' }
      );
      document.querySelectorAll('.reveal').forEach((el) => io.observe(el));

      // Footer yılı
      document.querySelectorAll('[data-year]').forEach((el) => {
        el.textContent = new Date().getFullYear();
      });
    },

    // Aktif bölümü tespit eder (nav alt çizgi vurgusu için)
    updateSection() {
      const atBottom =
        window.innerHeight + window.scrollY >= document.body.scrollHeight - 60;
      if (atBottom) {
        this.s = 'contact';
        return;
      }
      const ids = ['contact', 'blog', 'reviews', 'about', 'work', 'services', 'hero'];
      for (const id of ids) {
        const el = document.getElementById(id);
        if (el && window.scrollY >= el.offsetTop - 130) {
          this.s = id;
          return;
        }
      }
    },
  };
}
