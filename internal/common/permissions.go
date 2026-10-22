package common

// Permission is a capability a super-admin can grant to a user, either for
// every resource (empty ResourceRef) or for one channel or course.
type Permission string

const (
	PERMISSION_CHANNEL_EDIT       Permission = "classes.channel.edit"
	PERMISSION_CHANNEL_MEDIA      Permission = "classes.channel.media"
	PERMISSION_COURSE_EDIT        Permission = "classes.course.edit"
	PERMISSION_CLASSES_PUBLISH    Permission = "classes.publish"
	PERMISSION_CLASSES_SYNC       Permission = "classes.sync"
	PERMISSION_PERMISSIONS_MANAGE Permission = "permissions.manage"
)

var ALL_PERMISSIONS = []Permission{
	PERMISSION_CHANNEL_EDIT,
	PERMISSION_CHANNEL_MEDIA,
	PERMISSION_COURSE_EDIT,
	PERMISSION_CLASSES_PUBLISH,
	PERMISSION_CLASSES_SYNC,
	PERMISSION_PERMISSIONS_MANAGE,
}

type ResourceType string

const (
	RESOURCE_TYPE_ALL     ResourceType = ""
	RESOURCE_TYPE_CHANNEL ResourceType = "channel"
	RESOURCE_TYPE_COURSE  ResourceType = "course"
)

// ResourceRef names what a permission applies to. The zero value means
// "every resource"; a grant for a channel also covers that channel's courses.
type ResourceRef struct {
	Type ResourceType
	ID   string
}
