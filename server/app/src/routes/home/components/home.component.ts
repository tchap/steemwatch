import { Component, OnInit } from '@angular/core';
import { ROUTER_DIRECTIVES } from '@angular/router';

import { MessageService } from '../../../services/index';


@Component({
  moduleId: module.id,
  templateUrl: 'home.component.html',
  directives: [ROUTER_DIRECTIVES]
})
export class HomeComponent implements OnInit {

  constructor(private messageService: MessageService) {}

  ngOnInit() {
    this.messageService.hideMessage();
  }
}
