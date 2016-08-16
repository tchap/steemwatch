import { Routes, RouterModule } from '@angular/router';

import { HomeComponent }          from './routes/home/index';
import { EventsComponent }        from './routes/events/index';
import { EventStreamComponent }   from './routes/eventstream/index';
import { NotificationsComponent } from './routes/notifications/index';
import { ProfileComponent}        from './routes/profile/index';


export const routes: Routes = [
  {path: 'home',          component: HomeComponent},
  {path: 'events',        component: EventsComponent},
  {path: 'eventstream',   component: EventStreamComponent},
  {path: 'notifications', component: NotificationsComponent},
  {path: 'profile',       component: ProfileComponent}
];

export const routing = RouterModule.forRoot(routes);
