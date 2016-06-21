import { Component, OnInit } from '@angular/core';

import { MessageService } from '../../../services/index';

import { SlackComponent } from './slack.component';


@Component({
  moduleId: module.id,
  templateUrl: 'notifications.component.html',
  directives: [SlackComponent]
})
export class NotificationsComponent implements OnInit {

  constructor(private messageService: MessageService) {}

  ngOnInit() {
    this.messageService.hideMessage();
  }
}
