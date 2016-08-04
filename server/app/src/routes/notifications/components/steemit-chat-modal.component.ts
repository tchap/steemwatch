import { Component, Input, ViewChild } from '@angular/core';

import { HTTP_PROVIDERS } from '@angular/http';

import {
  REACTIVE_FORM_DIRECTIVES,
  FormGroup,
  FormControl,
  FormBuilder
} from '@angular/forms';

import { Modal } from 'angular2-modal';

import { SteemitChatService }  from '../services/steemit-chat.service';
import { SteemitChatSettings } from '../models/steemit-chat.model';


@Component({
  moduleId: module.id,
  selector: 'steemit-chat-modal-body',
  templateUrl: 'steemit-chat-modal.component.html',
  styleUrls: ['steemit-chat-modal.component.css'],
  directives: [REACTIVE_FORM_DIRECTIVES],
  providers: [HTTP_PROVIDERS, SteemitChatService]
})
class SteemitChatModalWindow {

  @ViewChild('closeButton') closeButton;

  model = {username: '', password: ''};

  processing:   boolean;
  errorMessage: string;

  form: FormGroup;

  constructor(
    private formBuilder: FormBuilder,
    private chatService: SteemitChatService
  ) {}

  onSubmit() {
    this.processing = true;
    this.errorMessage = null;

    const username = this.model.username;
    const password = this.model.password;

    this.chatService.logon(username, password)
      .subscribe(
        (creds) => this.chatService.store(username, creds)
          .subscribe(
            () => {
              this.onSuccess({
                username:  username,
                userID:    creds.userID,
                authToken: creds.authToken
              });
            },
            (err) => {
              this.chatService.logoff(creds)
                .subscribe(
                  () => {
                    this.onError(err);
                  },
                  (ex) => {
                    this.onError(err);
                    console.error(ex);
                  }
                );
            }
          ),
        (err) => this.onError(err)
      );
  }

  private closeModal() : void {
    setTimeout(() => {
      const evt = new MouseEvent('click', {bubbles: true});
      this.closeButton.nativeElement.dispatchEvent(evt);
    }, 0);
  }

  private resetModel() : void {
    this.model = {username: '', password: ''};
  }

  private onSuccess(settings: SteemitChatSettings) : void {
    this.processing = false;
    this.closeModal();
    this.resetModel();
  }

  private onError(err) : void {
    this.processing = false;
    this.errorMessage = (err.status ?
                         `${err.status} ${err.text()}` :
                         `${err.message || err}`);
  }
}


@Component({
  moduleId: module.id,
  selector: 'steemit-chat-modal',
  template: '<div class="modal"></div>',
  directives: [SteemitChatModalWindow]
})
export class SteemitChatModalComponent {

  @Input() onConnected: (settings: SteemitChatSettings) => void;

  constructor(
    private modal: Modal
  ) {}

  open() {
    return this.modal.open(SteemitChatModalWindow);
  }
}
