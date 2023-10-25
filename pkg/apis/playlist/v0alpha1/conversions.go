package v0alpha1

import (
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/grafana/grafana/pkg/services/playlist"
	"github.com/grafana/grafana/pkg/setting"
)

type namespaceMapper = func(orgId int64) string

func orgNamespaceMapper(orgId int64) string {
	if orgId == 1 {
		return "default"
	}
	return fmt.Sprintf("org-%d", orgId)
}

func getNamespaceMapper(cfg *setting.Cfg) namespaceMapper {
	if cfg.StackID != "" {
		return func(orgId int64) string { return "stack-" + cfg.StackID }
	}
	return orgNamespaceMapper
}

func convertToK8sResource(v *playlist.PlaylistDTO, namespacer namespaceMapper) *Playlist {
	spec := Spec{
		Title:    v.Name,
		Interval: v.Interval,
	}
	for _, item := range v.Items {
		spec.Items = append(spec.Items, Item{
			Type:  ItemType(item.Type),
			Value: item.Value,
		})
	}
	return &Playlist{
		ObjectMeta: metav1.ObjectMeta{
			Name:              v.Uid,
			UID:               types.UID(v.Uid),
			ResourceVersion:   fmt.Sprintf("%d", v.UpdatedAt),
			CreationTimestamp: metav1.NewTime(time.UnixMilli(v.CreatedAt)),
			Namespace:         namespacer(v.OrgID),
		},
		Spec: spec,
	}
}

func convertToLegacyUpdateCommand(p *Playlist, orgId int64) (*playlist.UpdatePlaylistCommand, error) {
	spec := p.Spec
	cmd := &playlist.UpdatePlaylistCommand{
		UID:      p.Name,
		Name:     spec.Title,
		Interval: spec.Interval,
		OrgId:    orgId,
	}
	for _, item := range spec.Items {
		if item.Type == ItemTypeDashboardById {
			return nil, fmt.Errorf("unsupported item type: %s", item.Type)
		}
		cmd.Items = append(cmd.Items, playlist.PlaylistItem{
			Type:  string(item.Type),
			Value: item.Value,
		})
	}
	return cmd, nil
}
