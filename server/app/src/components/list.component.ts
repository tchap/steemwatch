import { Component, Input, ViewChild, OnInit, AfterViewChecked } from '@angular/core';
import { Http, Headers }                                         from '@angular/http';

import { Observable } from 'rxjs/Observable';

import { CookieService } from 'angular2-cookie/core';


@Component({
  moduleId: module.id,
  selector: "list",
  templateUrl: "list.component.html"
})
export class ListComponent implements OnInit, AfterViewChecked {

  @Input() path: string[];

  @ViewChild('input') input;

  model: string[];
  userInput: string = '';
  disabled: boolean = true;
  errorMessage: string = '';

  private focus: boolean;

  constructor(
    private http:    Http,
    private cookies: CookieService
  ) {}

  ngOnInit() {
    this.load();
  }

  ngAfterViewChecked() {
    if (this.focus) {
      this.input.nativeElement.focus();
      this.focus = false;
    }
  }

  load() {
    // Send the API call.
    const url = `/api/${this.path.join('/')}`;

    const headers = new Headers({
      'X-CSRF-Token': this.cookies.get('csrf')
    });
    
    this.http.get(url, {headers})
      .subscribe(
        (res) => {
          this.model = res.json();
          this.model.sort();
          this.disabled = false;
        },
        (err) => {
          if (err.status) {
            this.errorMessage = `${err.status} ${err.text()}`;
          } else {
            this.errorMessage = err.message || err;
          }
          this.disabled = false;
        }
      );
  }

  add() {
    // Make sure here as well that nothing can be done on disabled.
    if (this.disabled) {
      return;
    }

    // Do nothing in case the input is empty.
    if (this.userInput === '') {
      return;
    }

    // Remove the leading @ in case it is present.
    this.userInput = this.userInput.replace('@', '');

    // Make sure the value is not in the list yet.
    for (let item of this.model) {
      if (item === this.userInput) {
        return;
      }
    }

    // Disable user interactions.
    this.disabled = true;

    // Send the API call.
    const url = `/api/${this.path.join('/')}`;

    const headers = new Headers({
      'X-CSRF-Token': this.cookies.get('csrf')
    });
    
    this.http.post(url, this.userInput, {headers})
      .subscribe(
        () => {
          this.model.push(this.userInput);
          this.model.sort();
          this.userInput = '';
          this.disabled = false;
          this.focus = true;
        },
        (err) => {
          if (err.status) {
            this.errorMessage = `${err.status} ${err.text()}`;
          } else {
            this.errorMessage = err.message || err;
          }
          this.disabled = false;
        }
      );
  }

  remove(index: number) {
    // Make sure here as well that nothing can be done on disabled.
    if (this.disabled) {
      return;
    }

    // Get the selected item.
    const item = this.model[index];

    // Disable user interactions.
    this.disabled = true;

    // Send the API call.
    const url = `/api/${this.path.join('/')}/${item}`;

    const headers = new Headers({
      'X-CSRF-Token': this.cookies.get('csrf')
    });
 
    this.http.delete(url, {headers})
      .subscribe(
        () => {
          this.model.splice(index, 1);
          this.disabled = false;
        },
        (err) => {
          if (err.status) {
            this.errorMessage = `${err.status} ${err.text()}`;
          } else {
            this.errorMessage = err.message || err;
          }
          this.disabled = false;
        }
      );
  }
}
