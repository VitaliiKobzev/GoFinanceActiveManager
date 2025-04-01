const notificationConfig = {
  position: 'top-right', // 'top-right', 'top-left', 'bottom-right', 'bottom-left'
  animationDuration: 300, // Длительность анимации в ms
  autoClose: 5000, // Автозакрытие через указанное время (ms), 0 - не закрывать
  maxNotifications: 5, // Максимальное количество одновременно показанных уведомлений
};

// Стили для уведомлений
const notificationStyles = `
  .notification-container {
    position: fixed;
    z-index: 1000;
    display: flex;
    flex-direction: column;
    gap: 10px;
    padding: 10px;
    pointer-events: none;
  }
  
  .notification-container.top-right {
    top: 20px;
    right: 20px;
  }
  
  .notification-container.top-left {
    top: 20px;
    left: 20px;
  }
  
  .notification-container.bottom-right {
    bottom: 20px;
    right: 20px;
  }
  
  .notification-container.bottom-left {
    bottom: 20px;
    left: 20px;
  }
  
  .notification {
    position: relative;
    padding: 15px 20px;
    background: #fff;
    border-radius: 5px;
    box-shadow: 0 3px 10px rgba(0, 0, 0, 0.1);
    opacity: 0;
    transform: translateY(20px);
    transition: all ${notificationConfig.animationDuration}ms ease;
    pointer-events: auto;
    cursor: pointer;
    max-width: 300px;
  }
  
  .notification.show {
    opacity: 1;
    transform: translateY(0);
  }
  
  .notification.success {
    background: #d4edda;
    color: #155724;
    border-left: 4px solid #28a745;
  }
  
  .notification.error {
    background: #f8d7da;
    color: #721c24;
    border-left: 4px solid #dc3545;
  }
  
  .notification.warning {
    background: #fff3cd;
    color: #856404;
    border-left: 4px solid #ffc107;
  }
  
  .notification.info {
    background: #d1ecf1;
    color: #0c5460;
    border-left: 4px solid #17a2b8;
  }
  
  .notification-title {
    font-weight: bold;
    margin-bottom: 5px;
  }
  
  .notification-close {
    position: absolute;
    top: 5px;
    right: 5px;
    background: none;
    border: none;
    font-size: 16px;
    cursor: pointer;
    opacity: 0.7;
  }
  
  .notification-close:hover {
    opacity: 1;
  }
`;

// Инициализация системы уведомлений
function initNotificationSystem() {
  // Добавляем стили в head документа
  const styleElement = document.createElement('style');
  styleElement.innerHTML = notificationStyles;
  document.head.appendChild(styleElement);
  
  // Создаем контейнер для уведомлений
  const container = document.createElement('div');
  container.className = `notification-container ${notificationConfig.position}`;
  container.id = 'notification-container';
  document.body.appendChild(container);
}

// Показать уведомление
function showNotification(options) {
  const container = document.getElementById('notification-container');
  if (!container) initNotificationSystem();
  
  const { title, message, type = 'info', autoClose = notificationConfig.autoClose } = options;
  
  // Создаем элемент уведомления
  const notification = document.createElement('div');
  notification.className = `notification ${type}`;
  
  // Добавляем заголовок, если он есть
  if (title) {
    const titleElement = document.createElement('div');
    titleElement.className = 'notification-title';
    titleElement.textContent = title;
    notification.appendChild(titleElement);
  }
  
  // Добавляем текст сообщения
  const messageElement = document.createElement('div');
  messageElement.textContent = message;
  notification.appendChild(messageElement);
  
  // Добавляем кнопку закрытия
  const closeButton = document.createElement('button');
  closeButton.className = 'notification-close';
  closeButton.innerHTML = '&times;';
  closeButton.addEventListener('click', () => closeNotification(notification));
  notification.appendChild(closeButton);
  
  // Ограничиваем количество уведомлений
  const notifications = container.querySelectorAll('.notification');
  if (notifications.length >= notificationConfig.maxNotifications) {
    closeNotification(notifications[0]);
  }
  
  // Добавляем уведомление в контейнер
  container.appendChild(notification);
  
  // Запускаем анимацию появления
  setTimeout(() => {
    notification.classList.add('show');
  }, 10);
  
  // Назначаем автоматическое закрытие, если нужно
  if (autoClose > 0) {
    setTimeout(() => {
      closeNotification(notification);
    }, autoClose);
  }
  
  // Закрытие по клику на само уведомление
  notification.addEventListener('click', (e) => {
    if (e.target !== closeButton) {
      closeNotification(notification);
    }
  });
  
  return notification;
}

// Закрыть уведомление
function closeNotification(notification) {
  if (!notification) return;
  
  notification.classList.remove('show');
  
  setTimeout(() => {
    notification.remove();
  }, notificationConfig.animationDuration);
}

// Вспомогательные функции для разных типов уведомлений
const notification = {
  success: (title, message, autoClose) => showNotification({ title, message, type: 'success', autoClose }),
  error: (title, message, autoClose) => showNotification({ title, message, type: 'error', autoClose }),
  warning: (title, message, autoClose) => showNotification({ title, message, type: 'warning', autoClose }),
  info: (title, message, autoClose) => showNotification({ title, message, type: 'info', autoClose }),
};

// Инициализируем систему при загрузке страницы
document.addEventListener('DOMContentLoaded', initNotificationSystem);

// Делаем функции доступными глобально
window.showNotification = showNotification;
window.notification = notification;

export { showNotification, notification, initNotificationSystem };