import { Routes } from '@angular/router';

export const routes: Routes = [
  {
    path: '',
    loadComponent: () =>
      import('../page/beranda').then((module) => module.Beranda),
  },
  {
    path: '**',
    redirectTo: '',
  },
];
