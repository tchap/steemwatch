import { Injectable } from '@angular/core';

import { Observable } from 'rxjs/Observable';
import 'rxjs/add/observable/interval';
import 'rxjs/add/operator/take';

import { ReconnectingWebSocket } from '../../../common/ReconnectingWebSocket';

import { EventModel } from '../models/event.model';


@Injectable()
export class EventStreamService {

  getSocket() : ReconnectingWebSocket {
    if (!window['WebSocket']) {
      throw new Error('WebSocket not available in this browser');
    }

    return new ReconnectingWebSocket('ws://localhost:8080/api/eventstream/ws', []);
  }
}
