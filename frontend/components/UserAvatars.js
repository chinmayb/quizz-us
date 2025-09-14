// UserAvatars Component
window.UserAvatars = {
  props: {
    users: {
      type: Array,
      required: true
    }
  },
  
  template: `
    <div class="avatar-group">
      <div class="avatar" v-for="user in displayUsers" :key="user.id">
        <img :src="user.avatar" :alt="user.name" v-if="user.avatar">
        <span v-else>{{ user.initials }}</span>
      </div>
      <div class="overflow-avatar" v-if="users.length > 3">
        +{{ users.length - 3 }}
      </div>
    </div>
  `,
  
  computed: {
    displayUsers() {
      return this.users.slice(0, 3);
    }
  }
};
