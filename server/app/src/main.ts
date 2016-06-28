import { bootstrap }                            from '@angular/platform-browser-dynamic';
import { disableDeprecatedForms, provideForms } from '@angular/forms';

import { AppComponent }         from './app.component';
import { APP_ROUTER_PROVIDERS } from './app.routes';


bootstrap(AppComponent, [
  disableDeprecatedForms(), provideForms(),
  APP_ROUTER_PROVIDERS
]);
