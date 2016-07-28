import { Component, OnInit, ChangeDetectorRef } from '@angular/core';

import { SteemitChatService }                    from '../services/steemit-chat.service';
import { SteemitChatSettings, SteemitChatModel } from '../models/steemit-chat.model';

import { SteemitChatModalComponent } from './steemit-chat-modal.component';


@Component({
  moduleId: module.id,
  selector: 'steemit-chat',
  templateUrl: 'steemit-chat.component.html',
  styleUrls: ['steemit-chat.component.css'],
  directives: [SteemitChatModalComponent],
  providers: [SteemitChatService]
})
export class SteemitChatComponent implements OnInit {

  model: SteemitChatModel;

  processing: boolean;
  errorMessage: string;

  constructor(
    private chatService: SteemitChatService,
    private ref:         ChangeDetectorRef
  ) {}

  ngOnInit() {
    this.chatService.load()
      .subscribe(
        (model) => this.model = model,
        (err) => this.errorMessage = `${err.message || err}`
      );
  }

  onConnected = (settings: SteemitChatSettings) : void => {
    this.model = {
      enabled: true,
      settings
    };
    this.ref.detectChanges();
  }

  setEnabled(enabled: boolean) : void {
    this.processing = true;
    this.errorMessage = null;

    this.chatService.setEnabled(enabled)
      .finally(() => this.processing = false)
      .subscribe(
        () => this.model.enabled = enabled,
        (err) => this.errorMessage = `${err.message || err}`
      );
  }

  disconnect() : void {
    this.processing = true;
    this.errorMessage = null;

    this.chatService.disconnect()
      .finally(() => this.processing = false)
      .subscribe(
        () => this.resetModel(),
        (err) => this.errorMessage = `${err.message || err}`
      );
  }

  private resetModel() : void {
    this.model = {
      enabled: false,
      settings: {
        username: ''
      }
    };
  }
}
