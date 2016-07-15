import { Injectable } from '@angular/core';

import { Observable } from 'rxjs/Observable';
import 'rxjs/add/observable/of';

import { Event } from '../../../interfaces';


const events : Event[] = [
  {
    id:          "account.updated",
    title:       "Account Updated",
    description: "User account was updated.",
    fields:      [
      {
        id:          "accounts",
        label:       "Accounts",
        description: "You will be notified when any of the following accounts is updated."
      }
    ]
  },
  {
    id:          "transfer.made",
    title:       "Transfer Made",
    description: "A transfer was made.",
    fields:      [
      {
        id:          "from",
        label:       "From",
        description: "You will be notified when a transfer is made from one of the following accounts."
      },
      {
        id:          "to",
        label:       "To",
        description: "You will be notified when a transfer is made to one of the following accounts."
      }
    ]
  },
  {
    id:          "user.mentioned",
    title:       "User Mentioned",
    description: "A user was mentioned in a story or a comment using @-mention notation.",
    fields:      [
      {
        id:          "users",
        label:       "Users",
        description: "You will be notified when any of the following users is @-mentioned on Steemit."
      },
      {
        id:          "authorBlacklist",
        label:       "Author Blacklist",
        description: "The notification is dropped when mentioned by one of the following authors."
      }
    ]
  },
  {
    id:          "story.published",
    title:       "Story Published",
    description: "A story was published",
    fields:      [
      {
        id:          "authors",
        label:       "Authors",
        description: "You will be notified when a story is published by one of the following authors."
      },
      {
        id:          "tags",
        label:       "Tags",
        description: "You will be notified when a story with one of the following tags is published."
      }
    ]
  },
  {
    id:          "story.voted",
    title:       "Story Voted",
    description: "A story vote was cast.",
    fields:      [
      {
        id:          "authors",
        label:       "Story Authors",
        description: "You will be notified when a story by one of the following authors is voted."
      },
      {
        id:          "voters",
        label:       "Story Voters",
        description: "You will be notified when a story vote is cast by one of the following voters."
      }
    ]
  },
  {
    id:          "comment.published",
    title:       "Comment Published",
    description: "A comment was published",
    fields:      [
      {
        id:          "authors",
        label:       "Authors",
        description: "You will be notified when a comment is published by one of the following authors."
      },
      {
        id:          "parentAuthors",
        label:       "Parent Authors",
        description: "You will be notified when a reply is published to a comment by one of the following authors."
      }
    ]
  },
  {
    id:          "comment.voted",
    title:       "Comment Voted",
    description: "A comment vote was cast.",
    fields:      [
      {
        id:          "authors",
        label:       "Comment Authors",
        description: "You will be notified when a comment by one of the following authors is voted."
      },
      {
        id:          "voters",
        label:       "Comment Voters",
        description: "You will be notified when a comment vote is cast by one of the following voters."
      }
    ]
  }
];


@Injectable()
export class EventsService {

  getEvents() : Observable<Event[]> {
    return Observable.of(events);
  }
}
