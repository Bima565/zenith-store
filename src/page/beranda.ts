import { HttpClient } from '@angular/common/http';
import {
  afterNextRender,
  Component,
  computed,
  DestroyRef,
  ElementRef,
  inject,
  signal,
  viewChild,
} from '@angular/core';

interface GameService {
  code: string;
  game: string;
  name: string;
  description?: string;
  status?: string;
  category?: string;
  icon?: string;
  grid?: string;
  price?: {
    basic: number;
  };
}

interface ServicesResponse {
  data: GameService[];
}

@Component({
  selector: 'app-beranda',
  standalone: true,
  templateUrl: './beranda.html',
})
export class Beranda {
  private readonly http = inject(HttpClient);
  private readonly destroyRef = inject(DestroyRef);
  private readonly popularTrack = viewChild<ElementRef<HTMLElement>>('popularTrack');

  protected readonly services = signal<GameService[]>([]);
  protected readonly searchQuery = signal('');
  protected readonly gameServices = computed(() =>
    this.services().filter((service) => !isNonGame(service)),
  );
  protected readonly voucherServices = computed(() =>
    this.services().filter((service) => isNonGame(service)),
  );
  protected readonly popularServices = computed(() => this.gameServices().slice(0, 16));
  protected readonly filteredGameServices = computed(() =>
    filterByQuery(this.gameServices(), this.searchQuery()),
  );
  protected readonly filteredVoucherServices = computed(() =>
    filterByQuery(this.voucherServices(), this.searchQuery()),
  );
  protected readonly loading = signal(true);
  protected readonly error = signal('');
  protected readonly autoSlidePaused = signal(false);

  private autoSlideTimer: number | null = null;

  constructor() {
    afterNextRender(() => {
      this.loadServices();
      this.startAutoSlide();
    });

    this.destroyRef.onDestroy(() => {
      this.stopAutoSlide();
    });
  }

  private loadServices(): void {
    this.http.get<ServicesResponse>('http://localhost:8080/api/services').subscribe({
      next: (response) => {
        const games = response.data
          .filter(
            (service, index, list) =>
              list.findIndex((item) => item.game === service.game) === index,
          )
          .map((service) => ({
            ...service,
            icon: service.icon || getGameIcon(service.game),
          }));
        this.services.set(games);
        this.loading.set(false);
      },
      error: (requestError: { error?: { message?: string }; message?: string }) => {
        this.error.set(
          requestError.error?.message ||
            requestError.message ||
            'Game tidak dapat dimuat dari server.',
        );
        this.loading.set(false);
      },
    });
  }

  protected getPoster(service: GameService): string {
    return service.grid || service.icon || '';
  }

  protected onPosterError(event: Event, fallbackIcon?: string): void {
    const img = event.target as HTMLImageElement | null;
    if (!img) {
      return;
    }
    if (fallbackIcon && img.dataset['fallbackApplied'] !== '1' && img.src !== fallbackIcon) {
      img.dataset['fallbackApplied'] = '1';
      img.src = fallbackIcon;
      return;
    }
    img.style.visibility = 'hidden';
  }

  protected onSearchInput(event: Event): void {
    const input = event.target as HTMLInputElement | null;
    this.searchQuery.set(input?.value ?? '');
  }

  protected formatPrice(price: number): string {
    return new Intl.NumberFormat('id-ID').format(price);
  }

  protected scrollPopular(direction: number): void {
    const container = this.popularTrack()?.nativeElement;
    if (!container) {
      return;
    }

    const step = this.getSlideStep(container) * 2;
    const maxScroll = container.scrollWidth - container.clientWidth;

    if (direction > 0 && container.scrollLeft >= maxScroll - 16) {
      container.scrollTo({ left: 0, behavior: 'smooth' });
    } else if (direction < 0 && container.scrollLeft <= 16) {
      container.scrollTo({ left: maxScroll, behavior: 'smooth' });
    } else {
      container.scrollBy({ left: direction * step, behavior: 'smooth' });
    }

    this.restartAutoSlide();
  }

  protected pauseAutoSlide(): void {
    this.autoSlidePaused.set(true);
  }

  protected resumeAutoSlide(): void {
    this.autoSlidePaused.set(false);
  }

  private startAutoSlide(): void {
    this.stopAutoSlide();
    this.autoSlideTimer = window.setInterval(() => {
      if (this.autoSlidePaused() || this.loading()) {
        return;
      }
      const container = this.popularTrack()?.nativeElement;
      if (!container || container.scrollWidth <= container.clientWidth) {
        return;
      }

      const maxScroll = container.scrollWidth - container.clientWidth;
      if (container.scrollLeft >= maxScroll - 20) {
        container.scrollTo({ left: 0, behavior: 'smooth' });
      } else {
        container.scrollBy({ left: this.getSlideStep(container), behavior: 'smooth' });
      }
    }, 3200);
  }

  private restartAutoSlide(): void {
    if (this.autoSlideTimer !== null) {
      this.startAutoSlide();
    }
  }

  private stopAutoSlide(): void {
    if (this.autoSlideTimer !== null) {
      window.clearInterval(this.autoSlideTimer);
      this.autoSlideTimer = null;
    }
  }

  private getSlideStep(container: HTMLElement): number {
    const firstCard = container.querySelector('article');
    if (!firstCard) {
      return 280;
    }
    const styles = window.getComputedStyle(container);
    const gap = Number.parseFloat(styles.columnGap || styles.gap || '20') || 20;
    return firstCard.getBoundingClientRect().width + gap;
  }
}

const NON_GAME_NAMES = new Set(['steam', 'playstation', 'xbox', 'google play', 'epic games']);

function isNonGame(service: GameService): boolean {
  if (service.category === 'voucher' || service.category === 'non-game') {
    return true;
  }
  return NON_GAME_NAMES.has(service.game.trim().toLowerCase());
}

function filterByQuery(list: GameService[], rawQuery: string): GameService[] {
  const query = rawQuery.trim().toLowerCase();
  if (!query) {
    return list;
  }
  return list.filter(
    (service) =>
      service.game.toLowerCase().includes(query) ||
      service.name.toLowerCase().includes(query) ||
      (service.description?.toLowerCase().includes(query) ?? false),
  );
}

function getGameIcon(game: string): string {
  const normalized = game.toLowerCase();
  const domain = normalized.includes('mobile legends')
    ? 'mobilelegends.com'
    : normalized.includes('free fire')
      ? 'ff.garena.com'
      : normalized.includes('pubg')
        ? 'pubg.com'
        : normalized.includes('valorant')
          ? 'playvalorant.com'
          : normalized.includes('genshin')
            ? 'genshin.hoyoverse.com'
            : normalized.includes('honkai')
              ? 'hsr.hoyoverse.com'
              : 'google.com';

  return `https://www.google.com/s2/favicons?domain=${domain}&sz=256`;
}
