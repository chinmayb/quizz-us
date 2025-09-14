// SidebarMenu Component
window.SidebarMenu = {
  props: {
    menuItems: {
      type: Array,
      required: true
    },
    questionRating: {
      type: Number,
      default: 0
    }
  },
  
  emits: ['select-menu-item', 'rate-question'],
  
  template: `
    <div class="menu">
      <div class="menu-header">
        <h3 class="menu-title">Quiz Options</h3>
      </div>
      
      <div class="menu-separator"></div>
      
      <div class="menu-section">
        <div class="menu-item" 
             v-for="item in menuItems" 
             :key="item.id"
             :class="{ active: item.active }"
             @click="$emit('select-menu-item', item)">
          <div class="menu-icon">
            <span v-if="item.icon">{{ item.icon }}</span>
          </div>
          <div class="menu-content">
            <div class="menu-label">{{ item.label }}</div>
            <div class="menu-description" v-if="item.description">{{ item.description }}</div>
          </div>
          <div class="menu-shortcut" v-if="item.shortcut">{{ item.shortcut }}</div>
        </div>
      </div>
      
      <div class="menu-separator"></div>
      
      <!-- Star Rating -->
      <div class="rating-section">
        <div class="rating-label">Rate this question:</div>
        <div class="star-rating">
          <span v-for="star in 5" 
                :key="star" 
                class="star"
                :class="{ filled: star <= questionRating }"
                @click="$emit('rate-question', star)">
            ★
          </span>
        </div>
      </div>
    </div>
  `
};
