import { Component, OnInit } from '@angular/core';

import { MessageService, Message } from '../services/message.service';


@Component({
  moduleId: module.id,
  selector: 'message',
  templateUrl: 'message.component.html',
})
export class MessageComponent implements OnInit {

  private message: Message;

  constructor(private service: MessageService) {}

  ngOnInit() {
    this.service.getMessageStream()
      .subscribe(
        (message: Message) => this.show(message),
        (err) => console.error('MessageComponent:', err),
        () => this.show({kind: 'danger', content: 'MessageService has crashed'})
      );
  }

  private show(message: Message) {
    if (message.kind === 'hide') {
      this.hide();
    } else {
      this.message = message;
    }
  }

  hide() {
    this.message = null;
  }
}
