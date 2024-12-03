#include "utils.h"

double Function(const double x) { return x * x * x; }

double CountIntegral(const double x, const double step) {
  return Function(x) * step;
}

double GetIntegral(const TTask &task) {
  LogWorker("starting integral calculation: " + task.AsString());

  double result = 0.0;
  for (double x = task.GetStart(); x < task.GetEnd(); x += task.GetStep()) {
    result += CountIntegral(x, task.GetStep());
  }
  return result;
}

void Discovery(const i32 discover_port) {
  i32 discover_sock = socket(AF_INET, SOCK_DGRAM, 0);
  if (discover_sock < 0) {
    LogWorker("Error socket creation");
    return;
  }

  i32 reuse = 1;
  if (setsockopt(discover_sock, SOL_SOCKET, SO_REUSEADDR, &reuse,
                 sizeof(reuse)) < 0) {
    LogWorker("Error setsockopt");
    close(discover_sock);
    return;
  }

  struct sockaddr_in discover_addr;
  discover_addr.sin_family = AF_INET;
  discover_addr.sin_addr.s_addr = INADDR_ANY;
  discover_addr.sin_port = htons(discover_port);

  if (bind(discover_sock, reinterpret_cast<struct sockaddr *>(&discover_addr),
           sizeof(discover_addr)) < 0) {
    LogWorker("Error bind discover_sock");
    close(discover_sock);
    return;
  }

  char buffer[256];
  while (true) {
    struct sockaddr_in master_addr;
    socklen_t addr_len = sizeof(master_addr);
    ssize_t n =
        recvfrom(discover_sock, buffer, sizeof(buffer) - 1, 0,
                 reinterpret_cast<struct sockaddr *>(&master_addr), &addr_len);
    if (n > 0) {
      buffer[n] = '\0';
      if (strcmp(buffer, "DISCOVER") == 0) {
        const char *response = "AVAILABLE";
        sendto(discover_sock, response, strlen(response), 0,
               reinterpret_cast<struct sockaddr *>(&master_addr), addr_len);
      }
    }
  }
}

void ListenTask(const i32 task_sock) {
  struct sockaddr_in client_addr;
  socklen_t addr_len = sizeof(client_addr);

  i32 client_sock = accept(
      task_sock, reinterpret_cast<struct sockaddr *>(&client_addr), &addr_len);
  if (client_sock < 0) {
    LogWorker("Error accept");
    return;
  }

  TTask task;
  ssize_t n = recv(client_sock, &task, sizeof(task), 0);

  if (n == sizeof(TTask)) {
    double result = GetIntegral(task);

    send(client_sock, &result, sizeof(result), 0);
  }

  close(client_sock);
}

void ProcessTasks(const i32 task_port) {
  i32 task_sock = socket(AF_INET, SOCK_STREAM, 0);
  if (task_sock < 0) {
    LogWorker("Error socket creation");
    return;
  }

  i32 reuse = 1;
  if (setsockopt(task_sock, SOL_SOCKET, SO_REUSEADDR, &reuse, sizeof(reuse)) <
      0) {
    LogWorker("Error setsockopt");
    close(task_sock);
    return;
  }

  struct sockaddr_in task_addr;
  task_addr.sin_family = AF_INET;
  task_addr.sin_addr.s_addr = INADDR_ANY;
  task_addr.sin_port = htons(task_port);

  if (bind(task_sock, reinterpret_cast<struct sockaddr *>(&task_addr),
           sizeof(task_addr)) < 0) {
    LogWorker("Error bind task_sock");
    close(task_sock);
    return;
  }

  if (listen(task_sock, 5) < 0) {
    LogWorker("Error while listen task_sock");
    close(task_sock);
    return;
  }

  while (true) {
    ListenTask(task_sock);
  }
}

void Run(const i32 discover_port, const i32 task_port) {
  std::thread discovery_thread(Discovery, discover_port);
  std::thread task_thread(ProcessTasks, task_port);

  discovery_thread.join();
  task_thread.join();
}

struct TInputParams {
  i32 discover_port = 0;
  i32 task_port = 0;

  TInputParams(int argc, char *argv[]) {
    try {
      if (argc != 3) {
        throw std::runtime_error("ERROR Wrong number of arguments");
      }
      discover_port = std::stoi(argv[1]);
      task_port = std::stoi(argv[2]);
    } catch (const std::exception &e) {
      std::cerr << e.what() << std::endl;
      exit(1);
    }
  }

  TString AsString() const {
    return "discover_port: " + std::to_string(discover_port) +
           ", task_port: " + std::to_string(task_port);
  }
};

int main(int argc, char *argv[]) {
  TInputParams params(argc, argv);

  LogWorker("starting " + params.AsString());

  Run(params.discover_port, params.task_port);
}
