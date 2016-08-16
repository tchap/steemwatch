import { NgModule }                         from '@angular/core';
import { BrowserModule }                    from '@angular/platform-browser';
import { FormsModule, ReactiveFormsModule } from '@angular/forms';
import { HttpModule }                       from '@angular/http';

import { AppComponent } from './app.component';

import { routing }                from './app.routing';
import { EventsComponent }        from './routes/events/index';
import { EventStreamComponent }   from './routes/eventstream/index';
import { HomeComponent }          from './routes/home/index';
import { NotificationsComponent } from './routes/notifications/index';
import { ProfileComponent}        from './routes/profile/index';

import { ListComponent }    from './components/list.component';
import { MessageComponent } from './components/message.component';


@NgModule({
  imports: [
    // Common
    BrowserModule,
    // Forms
    FormsModule,
    ReactiveFormsModule,
    // HTTP
    HttpModule,
    // Routing
    routing
  ],
  declarations: [
    // App
    AppComponent,
    // Routes
    EventsComponent,
    EventStreamComponent,
    HomeComponent,
    NotificationsComponent,
    ProfileComponent
  ],
  bootstrap: [AppComponent],
})
export class AppModule {}
