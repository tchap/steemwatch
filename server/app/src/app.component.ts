import { Component, OnInit }         from '@angular/core';
import { Location }                  from '@angular/common';
import { Router, ROUTER_DIRECTIVES } from '@angular/router';
import { HTTP_PROVIDERS }            from '@angular/http';

import { CookieService } from 'angular2-cookie/core';

import { ContextService, Context } from './services/context.service';
import { MessageComponent }        from './components/index';
import { MessageService }          from './services/index';


@Component({
  selector: 'app',
  templateUrl: '/app/src/app.component.html',
  providers: [
    ContextService,
    MessageService,
    HTTP_PROVIDERS,
    CookieService],
  directives: [ROUTER_DIRECTIVES, MessageComponent]
})
export class AppComponent implements OnInit {

  ctx: Context;

  constructor(
    private router: Router,
    private location: Location,
    private contextService: ContextService
  ) {
    this.ctx = contextService.getContext();
  }

  ngOnInit() {
    const subscription = this.router.events.subscribe(
      () => {
        if (!this.location.path()) {
          subscription.unsubscribe();
          this.router.navigate(['/home']);
        }
      }
    );
  }

  logout() {
    window.location.href = this.ctx.canonicalURL + '/logout';
  }
}
