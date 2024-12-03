#include "utils.h"

struct TInputParams {
public:
  TTask task;
  i32 broadcast_port;
  i32 task_port;
  TInstant timeout_sec;

public:
  TInputParams(int argc, char *argv[]) {
    try {
      const i32 params_count = 7;
      if (argc != params_count) {
        throw std::runtime_error("ERROR Wrong number of arguments, expected: " +
                                 std::to_string(params_count));
      }
      this->task =
          TTask(std::stod(argv[1]), std::stod(argv[2]), std::stod(argv[3]));
      this->broadcast_port = std::stoi(argv[4]);
      this->task_port = std::stoi(argv[5]);
      this->timeout_sec = std::stoi(argv[6]);
    } catch (const std::exception &e) {
      std::cerr << e.what() << std::endl;
      exit(1);
    }
  }

  TString AsString() const {
    return task.AsString() +
           ", broadcast_port: " + std::to_string(broadcast_port) +
           ", task_port: " + std::to_string(task_port) +
           ", timeout_sec: " + std::to_string(timeout_sec);
  }
};

class TContext {
private:
  i32 broadcast_port_ = 0;
  i32 task_port_ = 0;
  TInstant timeout_sec_ = 0;

public:
  TContext(const i32 broadcast_port, const i32 task_port)
      : broadcast_port_(broadcast_port), task_port_(task_port) {}

  TContext(const TInputParams &inputParams)
      : broadcast_port_(inputParams.broadcast_port),
        task_port_(inputParams.task_port),
        timeout_sec_(inputParams.timeout_sec) {}

  i32 GetBroadcastPort() const { return broadcast_port_; }

  i32 GetTaskPort() const { return task_port_; }

  TInstant GetTimeoutSec() const { return timeout_sec_; }
};

struct TServer {
  struct sockaddr_in addr;
  bool is_active = false;

  bool operator==(const TServer &other) const {
    return addr.sin_addr.s_addr == other.addr.sin_addr.s_addr;
  }

  TString AsString() const {
    return "Server: " + TString(inet_ntoa(this->addr.sin_addr)) +
           std::to_string(ntohs(this->addr.sin_port));
  }
};

class TServerList {
private:
  TVector<TServer> servers;
  mutable TMutex mutex;

public:
  TServerList() = default;

  void addServer(const TServer &server) {
    std::lock_guard<TMutex> lock(mutex);

    auto it = std::find_if(
        servers.begin(), servers.end(),
        [&server](const TServer &existing) { return existing == server; });

    if (it == servers.end()) {
      servers.push_back(server);
      LogClient(server.AsString());
    } else {
      it->is_active = true;
    }
  }

  size_t size() const {
    std::lock_guard<TMutex> lock(mutex);
    return servers.size();
  }

  TServer &operator[](size_t index) {
    std::lock_guard<TMutex> lock(mutex);
    return servers[index];
  }

  void PrintActiveServers() const {
    std::lock_guard<TMutex> lock(mutex);
    TString active_servers = "";
    for (size_t i = 0; i < servers.size(); ++i) {
      if (!servers[i].is_active) {
        continue;
      }
      active_servers += TString(inet_ntoa(servers[i].addr.sin_addr)) + " ";
    }
    LogClient("active servers: " + active_servers);
  }
};

TServerList serverList;

// returns > 0 if success
// returns 0 if failure
i32 ListenBroadcast(const i32 sock, TVector<char> &buffer,
                    struct sockaddr_in &serverAddr, socklen_t &addrLen,
                    const TContext &ctx) {
  static constexpr i32 SUCCESS = 1;
  static constexpr i32 FAILURE = 0;

  i32 res =
      recvfrom(sock, buffer.data(), buffer.size() - 1, 0,
               reinterpret_cast<struct sockaddr *>(&serverAddr), &addrLen);
  if (res < 0) {
    if (errno == EAGAIN || errno == EWOULDBLOCK) {
      return FAILURE;
    }
    LogClient("Error recv answer");
    return SUCCESS;
  }

  serverAddr.sin_port = htons(ctx.GetBroadcastPort());

  TServer server{serverAddr, true};
  serverList.addServer(server);

  return SUCCESS;
}

void GetServers(const TContext &ctx) {
  i32 sock = socket(AF_INET, SOCK_DGRAM, 0);
  if (sock < 0) {
    throw std::runtime_error("Error socket creation");
  }

  i32 broadcast = 1;
  if (setsockopt(sock, SOL_SOCKET, SO_BROADCAST, &broadcast,
                 sizeof(broadcast)) < 0) {
    close(sock);
    throw std::runtime_error("Error setsockopt");
  }

  struct sockaddr_in broadcastAddr;
  broadcastAddr.sin_family = AF_INET;
  broadcastAddr.sin_port = htons(ctx.GetBroadcastPort());
  broadcastAddr.sin_addr.s_addr = INADDR_BROADCAST;

  TString msg = "DISCOVER";
  if (sendto(sock, msg.c_str(), msg.length(), 0,
             reinterpret_cast<struct sockaddr *>(&broadcastAddr),
             sizeof(broadcastAddr)) < 0) {
    close(sock);
    throw std::runtime_error("Error sendto");
  }

  struct timeval tv {
    ctx.GetTimeoutSec(), 0
  };
  setsockopt(sock, SOL_SOCKET, SO_RCVTIMEO, &tv, sizeof(tv));

  TVector<char> buffer(256);
  struct sockaddr_in serverAddr;
  socklen_t addrLen = sizeof(serverAddr);

  while (true) {
    i32 res = ListenBroadcast(sock, buffer, serverAddr, addrLen, ctx);
    if (res == 0) {
      break;
    }
  }

  close(sock);
  serverList.PrintActiveServers();
}

double ProcessTask(TServer &server, const TTask &task, const TContext &ctx) {
  i32 sock = socket(AF_INET, SOCK_STREAM, 0);
  if (sock < 0) {
    server.is_active = false;
    throw std::runtime_error("Socket creation error");
  }

  struct timeval tv {
    ctx.GetTimeoutSec(), 0
  };
  setsockopt(sock, SOL_SOCKET, SO_RCVTIMEO, &tv, sizeof(tv));

  struct sockaddr_in taskAddr = server.addr;
  taskAddr.sin_port = htons(ctx.GetTaskPort());

  i32 flags = fcntl(sock, F_GETFL, 0);
  fcntl(sock, F_SETFL, flags | O_NONBLOCK);

  i32 conn_result = connect(
      sock, reinterpret_cast<struct sockaddr *>(&taskAddr), sizeof(taskAddr));

  if (conn_result < 0 && errno != EINPROGRESS) {
    server.is_active = false;
    close(sock);
    throw std::runtime_error("Connection error");
  }

  fd_set set;
  FD_ZERO(&set);
  FD_SET(sock, &set);

  conn_result = select(sock + 1, NULL, &set, NULL, &tv);

  if (conn_result <= 0) {
    server.is_active = false;
    close(sock);
    throw std::runtime_error("Connection error");
  }

  i32 so_error;
  socklen_t len = sizeof(so_error);
  getsockopt(sock, SOL_SOCKET, SO_ERROR, &so_error, &len);
  if (so_error != 0) {
    server.is_active = false;
    close(sock);
    throw std::runtime_error("Connection error");
  }

  fcntl(sock, F_SETFL, flags);

  if (send(sock, &task, sizeof(TTask), 0) < 0) {
    server.is_active = false;
    close(sock);
    throw std::runtime_error("Send error");
  }

  double result;
  if (recv(sock, &result, sizeof(result), 0) < 0) {
    server.is_active = false;
    close(sock);
    throw std::runtime_error("Recv error");
  }

  close(sock);
  return result;
}

void GiveTask(double &total_result, std::queue<size_t> &pending_tasks,
              TVector<std::pair<size_t, std::future<double>>> &futures,
              const TVector<TTask> &tasks, const TContext &ctx) {
  size_t server_count = serverList.size();
  LogClient("Tasks left: " + std::to_string(pending_tasks.size()));

  for (size_t i = 0; i < server_count && !pending_tasks.empty(); ++i) {
    TServer &server = serverList[i];
    if (!server.is_active) {
      continue;
    }

    size_t task_index = pending_tasks.front();
    pending_tasks.pop();

    futures.push_back(
        {task_index, std::async(std::launch::async,
                                [&server, &ctx, task = tasks[task_index]]() {
                                  return ProcessTask(server, task, ctx);
                                })});
  }

  for (auto &[task_index, future] : futures) {
    try {
      double result = future.get();
      total_result += result;
    } catch (const std::exception &e) {
      LogClient("Error by processing task");
      LogClient(e.what());

      pending_tasks.push(task_index);
      GetServers(ctx);
    }
  }
  futures.clear();

  if (!pending_tasks.empty()) {
    GetServers(ctx);
  }
}

double GiveTasks(const TVector<TTask> &tasks, const TContext &ctx) {
  double total_result = 0.0;
  TVector<std::pair<size_t, std::future<double>>> futures;
  std::queue<size_t> pending_tasks;

  for (size_t i = 0; i < tasks.size(); ++i) {
    pending_tasks.push(i);
  }

  while (!pending_tasks.empty()) {
    GiveTask(total_result, pending_tasks, futures, tasks, ctx);
  }

  return total_result;
}

TVector<TTask> SplitTasks(const TInputParams &params) {
  TVector<TTask> tasks;
  for (double x = params.task.GetStart(); x < params.task.GetEnd(); x += 1.0) {
    TTask task;
    task.SetStart(x);
    task.SetEnd(std::min(x + 1.0, params.task.GetEnd()));
    task.SetStep(params.task.GetStep());
    tasks.push_back(task);
  }
  return tasks;
}

int main(int argc, char *argv[]) {
  TInputParams params(argc, argv);
  TContext ctx(params);

  LogClient("Starting " + params.AsString());

  TVector<TTask> tasks = SplitTasks(params);

  LogClient("Splitted tasks: " + std::to_string(tasks.size()));

  GetServers(ctx);

  LogClient("Final result: " + std::to_string(GiveTasks(tasks, ctx)));
}
