import { Component }      from '@angular/core';

import { CookieService } from 'angular2-cookie/core';

import { ContextService, Context } from './services/context.service';
import { ProfileService }          from './services/profile.service';
import { MessageComponent }        from './components/index';
import { MessageService }          from './services/index';


@Component({
  selector: 'app',
  templateUrl: '/app/src/app.component.html',
  directives: [MessageComponent],
  providers: [
    ContextService,
    ProfileService,
    MessageService,
    CookieService
  ]
})
export class AppComponent {

  ctx: Context;

  constructor(
    private contextService: ContextService
  ) {
    this.ctx = contextService.getContext();
  }

  logout() {
    window.location.href = this.ctx.canonicalURL + '/logout';
  }
}
