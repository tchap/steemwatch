import { Component, Input, OnChanges } from '@angular/core';
import { Observable }                  from 'rxjs/Observable';

import { ListComponent } from '../../../components/index';

import * as Interfaces from '../../../interfaces';

import { EventsService } from '../services/events.service';


@Component({
  moduleId: module.id,
  selector: 'event-list-item',
  templateUrl: 'event-list-item.component.html',
  directives: [ListComponent]
})
export class EventListItemComponent implements OnChanges {

  @Input() model: Interfaces.Event;
  @Input() path: string[];

  lists: {path: string[], list: Interfaces.Field}[];

  ngOnChanges() {
    this.lists = this.model.fields.map(list => ({
      path: this.path.concat(list.id),
      list: list
    }));
  }
}
