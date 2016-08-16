import { Component, Input, ViewChild } from '@angular/core';

import 'rxjs/add/operator/finally';

import { SteemitChatService }  from '../services/steemit-chat.service';
import { SteemitChatSettings } from '../models/steemit-chat.model';


@Component({
  moduleId: module.id,
  selector: 'steemit-chat-modal',
  templateUrl: 'steemit-chat-modal.component.html',
  styleUrls: ['steemit-chat-modal.component.css'],
  providers: [SteemitChatService]
})
export class SteemitChatModalComponent {

  @Input() onConnected: (settings: SteemitChatSettings) => void;

  @ViewChild('closeButton') closeButton;

  model = {username: '', password: ''};

  processing:   boolean;
  errorMessage: string;

  constructor(
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
          // Drop the credentials when we are done.
          .finally(() => this.chatService.logoff(creds)
            .subscribe(
              () => {},
              (err) => console.error('failed to log out:', err)
            )
          )
          .subscribe(
            () => this.onSuccess({username}),
            (err) => this.onError(err)
          ),
        (err) => this.onError(err)
      );
  }

  private onSuccess(settings: SteemitChatSettings) : void {
    this.processing = false;

    this.closeModal();
    this.resetModel();

    if (this.onConnected) {
      this.onConnected(settings);
    }
  }

  private onError(err) : void {
    this.processing = false;
    this.errorMessage = (err.status ?
                         `${err.status} ${err.text()}` :
                         `${err.message || err}`);
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
}
