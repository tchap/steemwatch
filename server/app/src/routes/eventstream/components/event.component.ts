import {
  Component,
  OnInit,
  AfterViewInit,
  Input,
  ViewChild,
  ChangeDetectorRef
} from '@angular/core';

import {
  NgSwitch,
  NgSwitchCase,
  NgSwitchDefault
} from '@angular/common';

import { EventModel } from '../models/event.model';

import { AccountUpdatedEventComponent } from './event-account-updated.component';
import { TransferMadeEventComponent }   from './event-transfer-made.component';
import { UserMentionedEventComponent }   from './event-user-mentioned.component';
import { StoryPublishedEventComponent } from './event-story-published.component';
import { StoryVotedEventComponent } from './event-story-voted.component';
import { CommentPublishedEventComponent } from './event-comment-published.component';
import { CommentVotedEventComponent } from './event-comment-voted.component';


@Component({
  moduleId: module.id,
  selector: 'event',
  styleUrls: ['event.component.css'],
  templateUrl: 'event.component.html',
  directives: [
    NgSwitch,
    NgSwitchCase,
    NgSwitchDefault,
    AccountUpdatedEventComponent,
    TransferMadeEventComponent,
    UserMentionedEventComponent,
    StoryPublishedEventComponent,
    StoryVotedEventComponent,
    CommentPublishedEventComponent,
    CommentVotedEventComponent
  ]
})
export class EventComponent implements OnInit, AfterViewInit {

  @Input() accounts: string[];

  @ViewChild('ev') child;

  classMap = {
    'event': true
  };

  @Input() model: EventModel;

  constructor(
    private ref: ChangeDetectorRef
  ) {}

  ngOnInit() {
    this.classMap[this.model.kind.replace('.', '-')] = true;
  }

  ngAfterViewInit() {
    if (this.child.isRelated) {
      const related = this.accounts.some(account => this.child.isRelated(account));
      this.classMap['related'] = related;
      if (related) {
        // XXX: Not sure setTimeout is necessary.
        setTimeout(() => this.ref.detectChanges(), 0);
      }
    }
  }
}
