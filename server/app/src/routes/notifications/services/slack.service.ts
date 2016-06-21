import { Injectable }              from '@angular/core';
import { Http, Headers, Response } from '@angular/http';

import { Observable } from 'rxjs/Observable';
import 'rxjs/add/observable/of';

import { CookieService } from 'angular2-cookie/core';

import { SlackModel } from '../models/slack.model';


@Injectable()
export class SlackService {

  constructor(
    private http: Http,
    private cookies: CookieService
  ) {}

  load() : Observable<SlackModel> {
    // Send the API call.
    const url = `/api/notifiers/slack`;

    const headers = new Headers({
      'X-CSRF-Token': this.cookies.get('csrf')
    });
    
    return this.http.get(url, {headers})
      .map(res => <SlackModel>(res.json()));
  }

  save(model: SlackModel) : Observable<Response> {
    const url = '/api/notifiers/slack';

    const body = JSON.stringify(model);

    const headers = new Headers({
      'Content-Type': 'application/json',
      'X-CSRF-Token': this.cookies.get('csrf')
    });

    return this.http.put(url, body, {headers});
  }

  update(model) : Observable<Response> {
    const url = '/api/notifiers/slack';

    const body = JSON.stringify(model);

    const headers = new Headers({
      'Content-Type': 'application/json',
      'X-CSRF-Token': this.cookies.get('csrf')
    });

    return this.http.patch(url, body, {headers});
  }
}
