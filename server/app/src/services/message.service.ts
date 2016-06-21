import { Injectable } from '@angular/core';

import { Observable } from 'rxjs/Observable';
import { Subject }    from 'rxjs/Subject';


export interface Message {
  kind: string;
  content: string;
}


@Injectable()
export class MessageService {

  private $: Subject<Message> = new Subject<Message>();

  private message(kind: string, content: string) {
    this.$.next({kind, content});
  }

  success(message: string) {
    this.message('success', message);
  }

  info(message: string) {
    this.message('info', message);
  }

  warning(message: string) {
    this.message('warning', message);
  }

  danger(message: string) {
    this.message('danger', message);
  }

  error(err: Error) {
    this.danger(err.message);
    console.error(err);
  }

  hideMessage() {
    this.message('hide', '');
  }

  getMessageStream() : Observable<Message> {
    return this.$;
  }
}
