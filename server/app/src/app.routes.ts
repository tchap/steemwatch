import { provideRouter, RouterConfig } from '@angular/router';

import { HomeComponent }          from './routes/home/index';
import { EventsComponent }        from './routes/events/index';
import { NotificationsComponent } from './routes/notifications/index';


export const routes: RouterConfig = [
  {path: 'home',          component:  HomeComponent},
  {path: 'events',        component:  EventsComponent},
  {path: 'notifications', component:  NotificationsComponent}
];

export const APP_ROUTER_PROVIDERS = [
  provideRouter(routes)
];
