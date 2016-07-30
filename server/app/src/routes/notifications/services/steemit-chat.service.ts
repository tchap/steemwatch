import { Injectable }              from '@angular/core';
import { Http, Headers, Response } from '@angular/http';

import { Observable } from 'rxjs/Observable';

import { CookieService } from 'angular2-cookie/core';

import { SteemitChatModel } from '../models/steemit-chat.model';


@Injectable()
export class SteemitChatService {

  constructor(
    private http: Http,
    private cookies: CookieService
  ) {}

  load() : Observable<SteemitChatModel> {
    // Send the API call.
    const url = `/api/notifiers/steemit-chat`;

    const headers = new Headers({
      'X-CSRF-Token': this.cookies.get('csrf')
    });
    
    return this.http.get(url, {headers})
      .map(res => {
        const payload = res.json();
        payload.enabled = payload.enabled || false;
        payload.settings = payload.settings || {};
        return <SteemitChatModel>payload;
      })
  }
}
