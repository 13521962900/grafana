import { FolderDTO } from 'app/types';
import { NavModelItem, NavModel } from '@grafana/data';

export function buildNavModel(folder: FolderDTO): NavModelItem {
  return {
    icon: 'fa fa-folder-open',
    id: 'manage-folder',
    subTitle: 'Manage folder dashboards & permissions',
    url: '',
    text: folder.title,
    breadcrumbs: [{ title: 'Dashboards', url: 'dashboards' }],
    children: [
      {
        active: false,
        icon: 'uil uil-th',
        id: `folder-dashboards-${folder.uid}`,
        text: 'Dashboards',
        url: folder.url,
      },
      {
        active: false,
        icon: 'uil uil-clock-nine',
        id: `folder-permissions-${folder.uid}`,
        text: 'Permissions',
        url: `${folder.url}/permissions`,
      },
      {
        active: false,
        icon: 'icon icon-setting',
        id: `folder-settings-${folder.uid}`,
        text: 'Settings',
        url: `${folder.url}/settings`,
      },
    ],
  };
}

export function getLoadingNav(tabIndex: number): NavModel {
  const main = buildNavModel({
    id: 1,
    uid: 'loading',
    title: 'Loading',
    url: 'url',
    canSave: false,
    version: 0,
  });

  main.children[tabIndex].active = true;

  return {
    main: main,
    node: main.children[tabIndex],
  };
}
