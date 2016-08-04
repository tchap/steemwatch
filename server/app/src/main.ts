import { bootstrap }                            from '@angular/platform-browser-dynamic';
import { disableDeprecatedForms, provideForms } from '@angular/forms';

import { MODAL_BROWSER_PROVIDERS } from 'angular2-modal/platform-browser';

import { AppComponent }         from './app.component';
import { APP_ROUTER_PROVIDERS } from './app.routes';


bootstrap(AppComponent, [
  disableDeprecatedForms(), provideForms(),
  APP_ROUTER_PROVIDERS,
  ...MODAL_BROWSER_PROVIDERS
]);
