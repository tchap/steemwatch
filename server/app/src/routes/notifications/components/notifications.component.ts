import { Component, OnInit } from '@angular/core';

import { MessageService } from '../../../services/index';

import { SlackComponent }       from './slack.component';
import { SteemitChatComponent } from './steemit-chat.component';


@Component({
  moduleId: module.id,
  templateUrl: 'notifications.component.html',
  directives: [SlackComponent, SteemitChatComponent]
})
export class NotificationsComponent implements OnInit {

  constructor(private messageService: MessageService) {}

  ngOnInit() {
    this.messageService.hideMessage();
  }
}
