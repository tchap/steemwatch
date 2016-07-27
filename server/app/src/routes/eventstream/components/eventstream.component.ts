import { Component, OnInit, OnDestroy } from '@angular/core';

import { Observable } from 'rxjs/Observable';
import 'rxjs/add/observable/timer';
import 'rxjs/add/operator/buffer';
import 'rxjs/add/operator/do';

import { ReconnectingWebSocket } from '../../../common/ReconnectingWebSocket';

import { MessageService } from '../../../services/message.service';
import { ProfileService } from '../../../services/profile.service';

import { StatusComponent } from './status.component';
import { EventComponent }  from './event.component';

import { EventStreamService } from '../services/eventstream.service';
import { EventModel }         from '../models/event.model';


const MAX_FEED_SIZE = 10000;
const DESKTOP_NOTIFICATIONS_INTERVAL = 5 * 60 * 1000; // 1 minute


declare var Notification: any;


@Component({
  moduleId: module.id,
  templateUrl: 'eventstream.component.html',
  styleUrls: ['eventstream.component.css'],
  directives: [StatusComponent, EventComponent],
  providers: [EventStreamService]
})
export class EventStreamComponent implements OnInit, OnDestroy {

  model:    EventModel[];
  accounts: string[] = [];

  socket: ReconnectingWebSocket;

  constructor(
    private streamService: EventStreamService,
    private profileService: ProfileService,
    private messageService: MessageService
  ) {}

  ngOnInit() {
    this.messageService.hideMessage();

    this.profileService.getAccounts()
      .subscribe(
        (accounts) => this.accounts = accounts,
        (err) => this.messageService.error(err)
      );

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
        if (this.model.length > MAX_FEED_SIZE) {
          this.model = this.model.splice(0, MAX_FEED_SIZE);
        }
      });

    this.socket.errors
      .subscribe(err => console.error('WebSocket', err));

    this.connectDesktopNotifications(this.socket.messages);

    this.socket.connect();
  }

  ngOnDestroy() {
    this.socket.close();
  }

  private connectDesktopNotifications(messages: Observable<MessageEvent>) : void {
    // Make sure notifications are available.
    if (!('Notification' in window)) {
      this.messageService.warning('Desktop notifications are not supported in this browser');
      return;
    }

    let notification;
    const triggerNotification = (numEvents) => {
      if (notification) {
        notification.close();
        notification = null;
      }

      notification = new Notification('SteemWatch Event Stream', {
        icon: 'https://steemit.com/images/favicons/favicon-128.png',
        body: `${numEvents} new ${numEvents === 1 ? 'event' : 'events'} received`,
        tag:  'steemwatch'
      });
    };

    // Start notifications if allowed.
    let count = 0;
    let freeToGo = true;
    let dirty = false;

    const onEvent = () => {
      count++;

      if (freeToGo) {
        triggerNotification(count);
        count = 0;

        freeToGo = false;
        dirty = false;

        Observable.timer(DESKTOP_NOTIFICATIONS_INTERVAL)
          .subscribe(() => {
            freeToGo = true;
            if (dirty) {
              count--;
              setTimeout(onEvent, 0);
            }
          });
      } else {
        dirty = true;
      }
    };

    const startNotifications = () => messages.subscribe(onEvent);

    if (Notification.permission === 'granted') {
      startNotifications();
      return;
    }

    if (Notification.permission !== 'denied') {
      Notification.requestPermission(permission => {
        if (permission === 'granted') {
          startNotifications();
        }
      })
    }
  }
}
