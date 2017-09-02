import { Injectable }              from '@angular/core';
import { Http, Headers, Response } from '@angular/http';

import { Observable } from 'rxjs/Observable';
import 'rxjs/add/observable/of';

import { CookieService } from 'angular2-cookie/core';

import { DiscordModel } from '../models/discord.model';


@Injectable()
export class DiscordService {

  constructor(
    private http: Http,
    private cookies: CookieService
  ) {}

  load() : Observable<DiscordModel> {
    // Send the API call.
    const url = `/api/notifiers/discord`;

    const headers = new Headers({
      'X-CSRF-Token': this.cookies.get('csrf')
    });
    
    return this.http.get(url, {headers})
      .map(res => <DiscordModel>(res.json()));
  }

  update(model: any) : Observable<Response> {
    const url = '/api/notifiers/discord';

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
    const url = '/api/notifiers/discord';

    const headers = new Headers({
      'X-CSRF-Token': this.cookies.get('csrf')
    });

    return this.http.delete(url, {headers});
  }
}
