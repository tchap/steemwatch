import { enableProdMode }         from '@angular/core';
import { platformBrowserDynamic } from '@angular/platform-browser-dynamic';

import { AppModule } from './app.module';

import { Context } from './services/context.service';


declare var ctx: Context;
setAngularMode(ctx.env);


platformBrowserDynamic().bootstrapModule(AppModule);


function setAngularMode(env: string) : void {
  if (env !== 'development') {
    console.log(`Environment is set to '${env}', enabling Angular 2 production mode...`);
    enableProdMode();
  } else {
    console.log(`Environment is set to '${env}', staying in Angular 2 development mode...`);
  }
}
