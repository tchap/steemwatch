import { Injectable } from '@angular/core';

import { Observable } from 'rxjs/Observable';
import 'rxjs/add/observable/interval';
import 'rxjs/add/operator/take';

import { ContextService } from '../../../services/index';

import { ReconnectingWebSocket } from '../../../common/ReconnectingWebSocket';

import { EventModel } from '../models/event.model';


@Injectable()
export class EventStreamService {

  constructor(
    private contextService: ContextService
  ) {}

  getSocket() : ReconnectingWebSocket {
    if (!window['WebSocket']) {
      throw new Error('WebSocket not available in this browser');
    }

    const canonicalURL = this.contextService.getContext().canonicalURL.replace(/^http/, 'ws');
    return new ReconnectingWebSocket(canonicalURL + '/api/eventstream/ws', []);
  }
}
