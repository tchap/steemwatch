import { Component, Input } from '@angular/core';

import { Event } from '../../../interfaces';

import { EventListItemComponent }  from './event-list-item.component';
import { ContentSubtreeComponent } from './content-subtree.component';


@Component({
  selector: 'event-list',
  template: `
    <div class="event-list">
      <event-list-item *ngFor="let event of model"
        [model]="event"
        [path]="['events', event.id]"
      >
      </event-list-item>

      <content-subtree></content-subtree>
    </div>
  `,
  directives: [EventListItemComponent, ContentSubtreeComponent]
})
export class EventListComponent {

  @Input() model: Event[]
}
