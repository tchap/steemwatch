import { Component, Input, ViewChild, OnInit } from '@angular/core';

import { MessageService } from '../../../services/index';

import { DiscordService } from '../services/discord.service';
import { DiscordModel }   from '../models/discord.model';


@Component({
  moduleId: module.id,
  selector: 'discord',
  templateUrl: 'discord.component.html',
  styleUrls: ['discord.component.css'],
  providers: [DiscordService]
})
export class DiscordComponent implements OnInit {

  model: DiscordModel;

  processing: boolean;
  errorMessage: string;

  constructor(
    private discordService: DiscordService,
    private messageService: MessageService
  ) {}

  ngOnInit() {
    this.discordService.load()
      .subscribe(
        model => {
          this.model = model;
        },
        err => this.messageService.error(err)
      );
  }

  setEnabled(enabled: boolean) : void {
    this.processing = true;
    this.errorMessage = null;

    this.discordService.setEnabled(enabled)
      .finally(() => this.processing = false)
      .subscribe(
        () => this.model.enabled = enabled,
        (err) => this.errorMessage = `${err.message || err}`
      );
  }

  disconnect() : void {
    this.processing = true;
    this.errorMessage = null;

    this.discordService.disconnect()
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
        startToken: this.model.settings.startToken,
        username: null,
      }
    };
  }
}
