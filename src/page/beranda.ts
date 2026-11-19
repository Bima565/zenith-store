import { HttpClient } from '@angular/common/http';
import { afterNextRender, Component, inject, signal } from '@angular/core';

interface GameService {
  code: string;
  game: string;
  name: string;
  description?: string;
  status?: string;
  icon?: string;
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
  protected readonly services = signal<GameService[]>([]);
  protected readonly loading = signal(true);
  protected readonly error = signal('');

  constructor() {
    afterNextRender(() => this.loadServices());
  }

  private loadServices(): void {
    this.http.get<ServicesResponse>('http://localhost:8080/api/services').subscribe({
      next: (response) => {
        const games = response.data
          .filter((service, index, list) =>
            list.findIndex((item) => item.game === service.game) === index,
          )
          .map((service) => ({
            ...service,
            icon: getGameIcon(service.game),
          }))
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

  return `https://www.google.com/s2/favicons?domain=${domain}&sz=128`;
}