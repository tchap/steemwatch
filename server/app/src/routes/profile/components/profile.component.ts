import { Component } from '@angular/core';

import { ListComponent } from '../../../components/list.component';


@Component({
  moduleId: module.id,
  templateUrl: 'profile.component.html',
  styleUrls: ['profile.component.css'],
  directives: [ListComponent]
})
export class ProfileComponent {}
