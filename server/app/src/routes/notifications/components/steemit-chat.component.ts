import { Component, OnInit } from '@angular/core';

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

  errorMessage: string;

  constructor(
    private chatService: SteemitChatService
  ) {}

  ngOnInit() {
    this.chatService.load()
      .subscribe(
        (model) => this.model = model,
        (err) => this.errorMessage = `Error: ${err.message || err}`
      );
  }

  onConnected(settings: SteemitChatSettings) : void {
    this.model = {
      settings,
      enabled: true
    };
  }
}
