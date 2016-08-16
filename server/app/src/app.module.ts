import { NgModule }      from '@angular/core';
import { BrowserModule } from '@angular/platform-browser';
import { HttpModule }    from '@angular/http';
import { FormsModule }   from '@angular/forms';

import { AppComponent } from './app.component';

import { routing }                from './app.routing';
import { EventsComponent }        from './routes/events/index';
import { EventStreamComponent }   from './routes/eventstream/index';
import { HomeComponent }          from './routes/home/index';
import { NotificationsComponent } from './routes/notifications/index';
import { ProfileComponent}        from './routes/profile/index';


@NgModule({
  declarations: [
    AppComponent,
    EventsComponent,
    EventStreamComponent,
    HomeComponent,
    NotificationsComponent,
    ProfileComponent
  ],
  imports:      [
    BrowserModule,
    // Router
    routing,
    // Forms
    FormsModule,
    // HTTP
    HttpModule
  ],
  bootstrap:    [AppComponent],
})
export class AppModule {}
