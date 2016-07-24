import {
  Component,
  OnChanges,
  Input
} from '@angular/core';

import {
  NgSwitch,
  NgSwitchCase,
  NgSwitchDefault
} from '@angular/common';

import { Observable } from 'rxjs/Observable';
import 'rxjs/add/operator/takeWhile';

import { State } from '../../../common/ReconnectingWebSocket';


interface Socket {
  state: Observable<State>;
  connect: () => void;
}


@Component({
  moduleId: module.id,
  selector: 'status',
  styleUrls: ['status.component.css'],
  templateUrl: 'status.component.html',
  directives: [NgSwitch, NgSwitchCase, NgSwitchDefault]
})
export class StatusComponent implements OnChanges {

  static classMapBase = {
    'status': true
  };

  @Input() socket: Socket;

  classMap: any;

  stateString:              string;
  reconnectIntervalSeconds: number;

  private countdown: any;

  ngOnChanges() {
    this.socket.state.subscribe(state => {
      const stateClass = this.stateClassName(state.readyState);
      this.stateString = stateClass.toUpperCase();

      this.classMap = Object.assign({}, StatusComponent.classMapBase);
      this.classMap[stateClass] = true;

      switch (state.readyState) {
        case WebSocket.CLOSED:
          const interval = state.metadata.reconnectInterval;
          if (interval) {
            const seconds = interval / 1000;
            this.reconnectIntervalSeconds = seconds;
            this.countdown = Observable.interval(1000)
              .map(v => seconds - v - 1)
              .takeWhile(seconds => seconds !== 0)
              .subscribe(
                (seconds) => this.reconnectIntervalSeconds = seconds
              );
          }
          break;

        case WebSocket.CONNECTING:
          this.stopCountdown();
          break;
      }
    });
  }

  reconnect() {
    this.stopCountdown();
    this.socket.connect();
  }

  private stopCountdown() : void {
    if (this.countdown) {
      this.countdown.unsubscribe();
      this.countdown = null;
    }
  }

  private stateClassName(state) : string {
    switch (state) {
      case WebSocket.CONNECTING:
        return 'connecting';

      case WebSocket.OPEN:
        return 'connected';

      case WebSocket.CLOSING:
        return 'closing';

      case WebSocket.CLOSED:
        return 'disconnected';
    }
  }
}
