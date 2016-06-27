import { Injectable }                from '@angular/core';
import { Http, Headers, Response }   from '@angular/http';

import { CookieService } from 'angular2-cookie/core';

import { Observable } from 'rxjs/Observable';

import { ContentModalModel } from '../models/content-modal.model';


@Injectable()
export class ContentDescendantPublishedService {

  constructor(
    private http: Http,
    private cookies: CookieService
  ) {}

  add(model: ContentModalModel) : Observable<Response> {
    const url = '/api/events/descendant.published/edit';

    const payload = {
      type: 'add',
      selector: {
        contentURL: model.rootURL,
        mode:       model.selectMode,
        depthLimit: model.depthLimit
      }
    };
    if (model.selectMode !== 'depthLimit') {
      delete payload.selector['depthLimit'];
    }

    const headers = new Headers({
      'X-CSRF-Token': this.cookies.get('csrf'),
      'Content-Type': 'application/json'
    });

    return this.http.post(url, JSON.stringify(payload), {headers});
  }

  remove(contentURL: string) : Observable<Response> {
    const url = '/api/events/descendant.published/edit';

    const payload = {
      type: 'remove',
      selector: {contentURL}
    };

    const headers = new Headers({
      'X-CSRF-Token': this.cookies.get('csrf'),
      'Content-Type': 'application/json'
    });

    return this.http.post(url, JSON.stringify(payload), {headers});
  }

  list() : Observable<ContentModalModel[]> {
    const url = '/api/events/descendant.published';

    const headers = new Headers({
      'X-CSRF-Token': this.cookies.get('csrf'),
    });

    return this.http.get(url, {headers})
      .map(resp => resp.json().map(item => ({
        rootURL:    item.contentURL,
        selectMode: item.mode,
        depthLimit: item.depthLimit
      })));
  }
}
