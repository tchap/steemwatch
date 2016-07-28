import { enableProdMode }                      from '@angular/core';
import { bootstrap }                            from '@angular/platform-browser-dynamic';
import { disableDeprecatedForms, provideForms } from '@angular/forms';

import { AppComponent }         from './app.component';
import { APP_ROUTER_PROVIDERS } from './app.routes';

import { Context } from './services/context.service';


declare var ctx: Context;

if (ctx.env !== 'development') {
  console.log(`Environment is set to '${ctx.env}', enabling Angular 2 production mode...`);
  enableProdMode();
} else {
  console.log(`Environment is set to '${ctx.env}', staying in Angular 2 development mode...`);
}


bootstrap(AppComponent, [
  disableDeprecatedForms(), provideForms(),
  APP_ROUTER_PROVIDERS
]);
