import { Component, Input } from '@angular/core';


@Component({
  moduleId: module.id,
  selector: 'event-story-voted',
  templateUrl: 'event-story-voted.component.html',
  styleUrls: ['event-story-voted.component.css']
})
export class StoryVotedEventComponent {

  @Input() model: any;

  isRelated(account: string) : boolean {
    return (this.model.author === account);
  }
}
