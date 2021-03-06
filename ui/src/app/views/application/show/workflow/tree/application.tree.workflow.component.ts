import {Component, Input} from '@angular/core';
import {WorkflowItem} from '../../../../../model/application.workflow.model';
import {Application} from '../../../../../model/application.model';
import {Project} from '../../../../../model/project.model';
import {Remote} from '../../../../../model/repositories.model';

@Component({
    selector: 'app-application-tree-workflow',
    templateUrl: './application.tree.workflow.html'
})
export class ApplicationTreeWorkflowComponent {

    @Input() project: Project;
    @Input() remotes: Array<Remote>;
    @Input() application: Application;
    @Input() workflowItems: Array<WorkflowItem>;
    @Input() orientation: string;
    @Input() applicationFilter: any;
    constructor() { }
}
