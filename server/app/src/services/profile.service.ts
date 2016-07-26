import { Injectable } from '@angular/core';
import { Http }       from '@angular/http';

import { Observable } from 'rxjs/Observable';


@Injectable()
export class ProfileService {

  constructor(
    private http: Http
  ) {}

  getAccounts() : Observable<string[]> {
    return this.http.get('/api/profile/accounts')
      .map(resp => <string[]>resp.json());
  }
}
