import { Injectable }              from '@angular/core';
import { Http, Headers, Response } from '@angular/http';

import { Observable } from 'rxjs/Observable';
import 'rxjs/add/observable/of';

import { CookieService } from 'angular2-cookie/core';

import { TelegramModel } from '../models/telegram.model';


@Injectable()
export class TelegramService {

  constructor(
    private http: Http,
    private cookies: CookieService
  ) {}

  load() : Observable<TelegramModel> {
    // Send the API call.
    const url = `/api/notifiers/telegram`;

    const headers = new Headers({
      'X-CSRF-Token': this.cookies.get('csrf')
    });
    
    return this.http.get(url, {headers})
      .map(res => <TelegramModel>(res.json()));
  }

  update(model: any) : Observable<Response> {
    const url = '/api/notifiers/telegram';

    const headers = new Headers({
      'Content-Type': 'application/json',
      'X-CSRF-Token': this.cookies.get('csrf')
    });

    const body = JSON.stringify(model);

    return this.http.patch(url, body, {headers});
  }

  setEnabled(enabled: boolean) : Observable<Response> {
    return this.update({enabled});
  }

  disconnect() : Observable<Response> {
    const url = '/api/notifiers/telegram';

    const headers = new Headers({
      'X-CSRF-Token': this.cookies.get('csrf')
    });

    return this.http.delete(url, {headers});
  }
}
