import { Component, OnInit } from '@angular/core';

import { MessageService } from '../../../services/index';

import { Event } from '../../../interfaces';

import { EventsService }  from '../services/events.service';

import { EventListComponent } from './event-list.component';


@Component({
  moduleId: module.id,
  templateUrl: 'events.component.html',
  providers: [EventsService],
  directives: [EventListComponent]
})
export class EventsComponent implements OnInit {

  events: Event[]

  constructor(private eventsService: EventsService, private messageService: MessageService) {}

  ngOnInit() {
    this.messageService.hideMessage();

    this.eventsService.getEvents()
      .subscribe(
        events => this.events = events,
        err => this.messageService.error(err)
      );
  }
}
