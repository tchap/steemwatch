import { Component, ViewChild, OnInit } from '@angular/core';
import { Http, Headers }                from '@angular/http';
import { FormGroup, FormControl }       from '@angular/forms';

import { CookieService } from 'angular2-cookie/core';

import { ContentDescendantPublishedService } from '../services/content-descendants.service';

import { ContentModalModel } from '../models/content-modal.model';


@Component({
  moduleId: module.id,
  selector: "content-subtree",
  templateUrl: "content-subtree.component.html",
  providers: [ContentDescendantPublishedService]
})
export class ContentSubtreeComponent implements OnInit {

  listModel: ContentModalModel[] = [];

  modalModel: ContentModalModel = this.newModalModel();

  form: FormGroup;

  @ViewChild('rootURL') rootURLChild;
  @ViewChild('closeButton') closeButton;

  saving: boolean;
  saveErrorMessage: string;
  removeErrorMessage: string;

  constructor(
    private http:    Http,
    private cookies: CookieService,
    private service: ContentDescendantPublishedService
  ) {}

  ngOnInit() {
    this.initForm();
    this.service.list()
      .subscribe(
        items => {
          this.listModel = items;
          this.sortList();
        },
        err => {
          if (err.status) {
            this.saveErrorMessage = `${err.status} ${err.text()}`;
          } else {
            this.saveErrorMessage = err.message || err;
          }
        }
      )
  }

  private initForm() {
    const validateRootURL = (c: FormControl) => {
      const CONTENT_REGEXP = /^https:\/\/steemit.com\/[^\/]+\/[^\/]+\/[^\/]+/;
      const isValid = this.rootURLChild.nativeElement.checkValidity() &&
        c.value && c.value.match(CONTENT_REGEXP);

      return isValid ? null : {
        validateURL: {
          valid: false
        }
      };
    };

    const validateDepthLimit = (c: FormControl) => {
      if (this.modalModel.selectMode !== 'depthLimit') {
        return null;
      }

      const isValid = c.value && parseInt(c.value, 10) >= 1;

      return isValid ? null : {
        validateDepthLimit: {
          valid: false
        }
      };
    };

    this.form = new FormGroup({
      rootURL:    new FormControl(),
      selectMode: new FormControl(),
      depthLimit: new FormControl()
    });

    /*
    this.form = this.formBuilder.group({
      rootURL: ['', validateRootURL],
      selectMode: [],
      depthLimit: ['', validateDepthLimit]
    });
   */

    this.modalModel = this.newModalModel();
  }

  modeChanged(mode) {
    this.form.controls['depthLimit'].updateValueAndValidity();
  }

  onSubmit() {
    this.saving = true;
    
    this.service.add(this.modalModel)
      .subscribe(
        () => {
          this.listModel.push(this.modalModel);
          this.sortList();
          this.modalModel = this.newModalModel();
          this.saving = false;
          this.saveErrorMessage = null;

          setTimeout(() => {
            const evt = new MouseEvent('click', {bubbles: true});
            this.closeButton.nativeElement.dispatchEvent(evt);
          }, 0);
        },
        (err) => {
          if (err.status) {
            this.saveErrorMessage = `${err.status} ${err.text()}`;
          } else {
            this.saveErrorMessage = err.message || err;
          }
          this.saving = false;
        }
      );
  }

  private newModalModel() : ContentModalModel {
    return {
      rootURL: '',
      selectMode: 'any',
      depthLimit: 1
    };
  }

  remove(i: number) {
    this.service.remove(this.listModel[i].rootURL)
      .subscribe(
        () => this.listModel.splice(i, 1),
        (err) => {
          if (err.status) {
            this.saveErrorMessage = `${err.status} ${err.text()}`;
          } else {
            this.saveErrorMessage = err.message || err;
          }
        }
      )
  }

  private sortList() {
    this.listModel.sort((a, b) => a.rootURL.localeCompare(b.rootURL));
  }
}
