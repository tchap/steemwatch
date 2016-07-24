import { Component, Input } from '@angular/core';


@Component({
  moduleId: module.id,
  selector: 'event-story-published',
  styleUrls: ['event-story-published.component.css'],
  templateUrl: 'event-story-published.component.html'
})
export class StoryPublishedEventComponent {

  @Input() model: any;
}
