import { Component, OnInit, OnDestroy } from '@angular/core';

import { Observable } from 'rxjs/Observable';

import { ReconnectingWebSocket } from '../../../common/ReconnectingWebSocket';

import { MessageService } from '../../../services/index';

import { StatusComponent } from './status.component';
import { EventComponent }  from './event.component';

import { EventStreamService } from '../services/eventstream.service';
import { EventModel }         from '../models/event.model';


@Component({
  moduleId: module.id,
  templateUrl: 'eventstream.component.html',
  styleUrls: ['eventstream.component.css'],
  directives: [StatusComponent, EventComponent],
  providers: [EventStreamService]
})
export class EventStreamComponent implements OnInit, OnDestroy {

  model: EventModel[];

  socket: ReconnectingWebSocket;

  constructor(
    private streamService: EventStreamService,
    private messageService: MessageService
  ) {}

  ngOnInit() {
    this.messageService.hideMessage();

    try {
      this.socket = this.streamService.getSocket();
    } catch (err) {
      this.messageService.error(err);
      return;
    }

    this.socket.messages
      .subscribe((ev) => {
        const event = JSON.parse(ev.data);
        this.model = this.model || [];
        this.model.unshift(event);
      });

    this.socket.errors
      .subscribe(err => console.error('WebSocket', err));

    this.socket.connect();
  }

  ngOnDestroy() {
    this.socket.close();
  }
}
