import { NgModule }      from '@angular/core';
import { BrowserModule } from '@angular/platform-browser';
import { FormsModule }   from '@angular/forms';

import { routing } from './app.routing';

import { AppComponent } from './app.component';


@NgModule({
  declarations: [AppComponent],
  imports:      [
    BrowserModule,
    // Router
    routing,
    // Forms
    FormsModule,
  ],
  bootstrap:    [AppComponent],
})
export class AppModule {}
