import { Component, OnInit } from '@angular/core';

import { SteemitChatService } from '../services/steemit-chat.service';
import { SteemitChatModel }   from '../models/steemit-chat.model';


@Component({
  moduleId: module.id,
  selector: 'steemit-chat',
  templateUrl: 'steemit-chat.component.html',
  styleUrls: ['steemit-chat.component.css'],
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
        model => this.model = model,
        err => this.errorMessage = err.message || err
      );
  }

  enable() : void {}
}
