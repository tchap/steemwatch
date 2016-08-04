import { Component, OnInit, ViewContainerRef } from '@angular/core';
import { Location }                            from '@angular/common';
import { Router, ROUTER_DIRECTIVES }           from '@angular/router';
import { HTTP_PROVIDERS }                      from '@angular/http';

import { CookieService } from 'angular2-cookie/core';

import { Modal, BS_MODAL_PROVIDERS } from 'angular2-modal/plugins/bootstrap';

import { APP_ROUTE_COMPONENTS } from './app.routes';

import { ContextService, Context } from './services/context.service';
import { ProfileService }          from './services/profile.service';
import { MessageComponent }        from './components/index';
import { MessageService }          from './services/index';


@Component({
  selector: 'app',
  templateUrl: '/app/src/app.component.html',
  directives: [ROUTER_DIRECTIVES, MessageComponent],
  providers: [
    ContextService,
    ProfileService,
    MessageService,
    HTTP_PROVIDERS,
    CookieService
  ],
  viewProviders: [...BS_MODAL_PROVIDERS],
  precompile: APP_ROUTE_COMPONENTS
})
export class AppComponent implements OnInit {

  ctx: Context;

  constructor(
    private router: Router,
    private location: Location,
    private contextService: ContextService,
    private modal: Modal,
    private viewContainer: ViewContainerRef
  ) {
    this.ctx = contextService.getContext();
    modal.defaultViewContainer = viewContainer;
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
