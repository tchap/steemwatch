import { provideRouter, RouterConfig } from '@angular/router';

import { HomeComponent }          from './routes/home/index';
import { EventsComponent }        from './routes/events/index';
import { EventStreamComponent }   from './routes/eventstream/index';
import { NotificationsComponent } from './routes/notifications/index';
import { ProfileComponent}        from './routes/profile/index';


export const routes: RouterConfig = [
  {path: 'home',          component:  HomeComponent},
  {path: 'events',        component:  EventsComponent},
  {path: 'eventstream',   component:  EventStreamComponent},
  {path: 'notifications', component:  NotificationsComponent},
  {path: 'profile',       component:  ProfileComponent}
];

export const APP_ROUTE_COMPONENTS = [
  HomeComponent,
  EventsComponent,
  EventStreamComponent,
  NotificationsComponent,
  ProfileComponent
];

export const APP_ROUTER_PROVIDERS = [
  provideRouter(routes)
];
