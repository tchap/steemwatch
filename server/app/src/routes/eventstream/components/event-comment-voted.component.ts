import { Component, Input } from '@angular/core';


@Component({
  moduleId: module.id,
  selector: 'event-comment-voted',
  templateUrl: 'event-comment-voted.component.html',
  styleUrls: ['event-comment-voted.component.css']
})
export class CommentVotedEventComponent {

  @Input() model: any;
}
